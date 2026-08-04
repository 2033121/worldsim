// Package themes 提供"题材知识卡片"：把题材该想哪些维度做成可插拔、可调用、可增删的 JSON 卡片。
//
// 卡片内容为**维度参考**（要素池：该世界"该想哪些维度"），而非叙事示例（不含具体人物/资产数值/情节）。
// —— 这是去硬编码的关键：Agent 提示词里不再写死"修仙=山村砍柴/都市=便利店"，改为按题材注入维度提示词。
//
// 参考 SillyTavern Lorebook（关键词触发 + 自包含条目）与 inkOS genres（物理文件 + 项目覆盖）。
// 加载顺序：内置 go:embed cards/*.json → 外部 <dataDir>/*.json（同 id 覆盖内置）。
package themes

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

//go:embed cards/*.json
var builtinCards embed.FS

// Dimensions 该题材下各维度的"要素池"（维度名清单，非具体数值/例子）。
type Dimensions struct {
	Assets []string `json:"assets"` // 资产/资源维度名（如 修仙["灵石","丹药","功法"]）
	Body   []string `json:"body"`   // 身体/精神状态维度名（如 修仙["灵力","伤势","心境"]）
	Self   []string `json:"self"`   // 主角自我/身份维度名（如 修仙["灵根资质","境界","寿元"]）
}

// Theme 一张题材知识卡片。
type Theme struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Aliases  []string   `json:"aliases"` // 识别用别名（含题材名近义词）
	Dim      Dimensions `json:"dimensions"`
	Locations []string  `json:"locations"`   // 常见地点类型
	TimeScale string    `json:"time_scale"`  // 时间尺度提示（不写死天数）
	LifeTexture []string `json:"life_texture"` // 生活质感提示（写"该想什么生活细节"，不给具体例子）
	EventsPool []string `json:"events_pool"` // 该题材典型事件方向（要素池）
	Taboos    []string `json:"taboos"`     // 串味红线（禁止借用哪些别题材的维度/质感/示例）
}

// Store 题材卡片仓库。
type Store struct {
	dataDir string // 外部卡片目录（wsdata/themes），可为空
	mu      sync.RWMutex
	cards   map[string]*Theme
}

// Load 加载内置 + 外部题材卡片。外部覆盖内置（同 id）；损坏文件跳过。
// dataDir 可为空（仅内置）。
func Load(dataDir string) *Store {
	s := &Store{dataDir: dataDir, cards: map[string]*Theme{}}
	// 内置
	if entries, err := builtinCards.ReadDir("cards"); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			b, err := builtinCards.ReadFile("cards/" + e.Name())
			if err != nil {
				continue
			}
			var t Theme
			if json.Unmarshal(b, &t) != nil || t.ID == "" {
				continue
			}
			s.cards[t.ID] = &t
		}
	}
	// 外部覆盖/新增
	if dataDir != "" {
		if entries, err := os.ReadDir(dataDir); err == nil {
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
					continue
				}
				b, err := os.ReadFile(filepath.Join(dataDir, e.Name()))
				if err != nil {
					continue
				}
				var t Theme
				if json.Unmarshal(b, &t) != nil || t.ID == "" {
					continue
				}
				s.cards[t.ID] = &t
			}
		}
	}
	return s
}

// List 返回全部卡片（按 id 排序）。
func (s *Store) List() []Theme {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Theme, 0, len(s.cards))
	for _, t := range s.cards {
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Find 按 id 查找卡片。
func (s *Store) Find(id string) *Theme {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if t, ok := s.cards[id]; ok {
		return t
	}
	return nil
}

// Detect 用 aliases/name/id 识别文本命中的题材卡片；无匹配返回 nil（走通用降级）。
func (s *Store) Detect(text string) *Theme {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	low := strings.ToLower(text)
	for _, t := range s.cards {
		if t.ID != "" && strings.Contains(low, strings.ToLower(t.ID)) {
			return t
		}
		if t.Name != "" && strings.Contains(low, strings.ToLower(t.Name)) {
			return t
		}
		for _, a := range t.Aliases {
			if a != "" && strings.Contains(low, strings.ToLower(a)) {
				return t
			}
		}
	}
	return nil
}

// Inject 返回 id 卡片的可注入提示词文本；无该卡片返回空串。
func (s *Store) Inject(id string) string {
	t := s.Find(id)
	if t == nil {
		return ""
	}
	return t.InjectText()
}

// InjectText 生成该题材的维度提示词（供 Agent prompt 注入）。只给维度参考，不给具体例子。
func (t *Theme) InjectText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "【本世界题材：%s】请只按这个题材的维度思考，不要借用其他题材的设定。\n", t.Name)
	if len(t.Dim.Assets) > 0 {
		fmt.Fprintf(&b, "· 资产/资源维度（可选的维度名，数值与具体拥有量由世界书和剧情决定）：%s\n", strings.Join(t.Dim.Assets, "/"))
	}
	if len(t.Dim.Body) > 0 {
		fmt.Fprintf(&b, "· 身体/精神维度（可选的维度名）：%s\n", strings.Join(t.Dim.Body, "/"))
	}
	if len(t.Dim.Self) > 0 {
		fmt.Fprintf(&b, "· 主角自我/身份维度（可选的维度名）：%s\n", strings.Join(t.Dim.Self, "/"))
	}
	if len(t.Locations) > 0 {
		fmt.Fprintf(&b, "· 常见地点类型：%s\n", strings.Join(t.Locations, "/"))
	}
	if t.TimeScale != "" {
		fmt.Fprintf(&b, "· 时间尺度：%s\n", t.TimeScale)
	}
	if len(t.LifeTexture) > 0 {
		fmt.Fprintf(&b, "· 生活质感（该想哪些生活细节，具体样子按本世界写）：%s\n", strings.Join(t.LifeTexture, "/"))
	}
	if len(t.EventsPool) > 0 {
		fmt.Fprintf(&b, "· 典型事件方向（要素池，具体事件按本世界写）：%s\n", strings.Join(t.EventsPool, "/"))
	}
	if len(t.Taboos) > 0 {
		fmt.Fprintf(&b, "· 串味红线（禁止借用）：%s\n", strings.Join(t.Taboos, "/"))
	}
	return strings.TrimRight(b.String(), "\n")
}

// Save 把卡片写入外部目录 <dataDir>/{id}.json（同 id 覆盖；新增）。供研究智能体沉淀题材卡片用。
func (s *Store) Save(card Theme) error {
	if strings.TrimSpace(card.ID) == "" {
		return fmt.Errorf("题材卡片 id 不能为空")
	}
	if s.dataDir == "" {
		return fmt.Errorf("未配置外部题材卡片目录")
	}
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(s.dataDir, card.ID+".json"), b, 0644); err != nil {
		return err
	}
	s.mu.Lock()
	s.cards[card.ID] = &card
	s.mu.Unlock()
	return nil
}