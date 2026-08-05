package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"worldsim/internal/bridge"
)

// PostWorldSeedNovel POST /api/world/seed-novel
//
// 从已模拟世界播种小说项目：读取世界数据（世界书 / 角色 / 势力 / 近期编年史事件），
// 生成小说项目的大纲 / 角色 / 世界观种子，写入 storys/{project_name}/。
// 全程零 LLM 调用 —— 世界设定/角色档案直接复用为种子，降低输入 token 与缓存未命中率。
//
// 请求体：{"world_id": "...", "project_name": "...", "language": "zh|en"}
// 返回：bridge.SeedResult（项目名 / 世界名 / 角色数 / 世界观数 / 开篇章节数 / 天数 / 是否复用）。
func (h *Handlers) PostWorldSeedNovel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorldID     string `json:"world_id"`
		ProjectName string `json:"project_name"`
		Language    string `json:"language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorReq(w, r, http.StatusBadRequest, "invalid_json")
		return
	}

	worldName := strings.TrimSpace(req.WorldID)
	if worldName == "" {
		h.writeErrorReq(w, r, http.StatusBadRequest, "world_id_required")
		return
	}

	projectName := strings.TrimSpace(req.ProjectName)
	if projectName == "" {
		h.writeErrorReq(w, r, http.StatusBadRequest, "missing_project_name")
		return
	}
	for _, c := range projectName {
		if c == '/' || c == '\\' || c == ':' || c == '*' || c == '?' || c == '"' || c == '<' || c == '>' || c == '|' {
			h.writeErrorReq(w, r, http.StatusBadRequest, "project_name_invalid_chars")
			return
		}
	}

	// 1. 读取世界数据（同时校验世界存在）
	worldData, err := bridge.ReadWorld(h.progDir, worldName)
	if err != nil {
		h.writeErrorReq(w, r, http.StatusNotFound, "world_not_found", worldName)
		return
	}

	// 2. 项目目录（已存在则拒绝，避免覆盖用户已有内容）
	projectDir := filepath.Join(h.storysDir(), projectName)
	if _, err := os.Stat(projectDir); err == nil {
		h.writeErrorReq(w, r, http.StatusConflict, "project_exists")
		return
	}

	// 3. 播种：创建项目 + config + 角色/世界观/组织/大纲种子
	result, err := bridge.SeedProjectFromWorld(projectDir, req.Language, worldData)
	if err != nil {
		h.writeErrorReq(w, r, http.StatusInternalServerError, "seed_novel_failed", err.Error())
		return
	}

	h.logger.InfoKey("log.seed_novel_done", result.WorldName, result.ProjectName,
		result.CharacterCount, result.WorldviewCount, result.OutlineChapterCount)
	h.writeJSON(w, http.StatusOK, result)
}