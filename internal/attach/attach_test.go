package attach

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

// makeDocx 构造一个最小可用的 .docx（zip 包，含 word/document.xml）。
func makeDocx(t *testing.T, bodyXML string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(bodyXML)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractDocx(t *testing.T) {
	body := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>修仙职场：给神仙发工资的HR</w:t></w:r></w:p>
    <w:p><w:r><w:t>第二段 设定说明</w:t></w:r></w:p>
  </w:body>
</w:document>`
	text, ok := extractDocx(makeDocx(t, body))
	if !ok {
		t.Fatal("期望成功提取 docx 文本")
	}
	if !strings.Contains(text, "修仙职场") || !strings.Contains(text, "第二段") {
		t.Errorf("提取文本不完整: %q", text)
	}
	if !strings.Contains(text, "\n") {
		t.Errorf("段落应换行: %q", text)
	}
	t.Logf("提取结果: %q", text)
}

func TestExtractDocx_NotZip(t *testing.T) {
	if _, ok := extractDocx([]byte("not a zip file")); ok {
		t.Error("非 zip 内容不应提取成功")
	}
}

func TestExtractText_DocxExtractable(t *testing.T) {
	// 通过 Save 走完整链路，验证 .docx 被标记为可行提取
	dir := t.TempDir()
	s := NewStore(dir)
	att, err := s.Save("设定.docx", makeDocx(t, `<w:document><w:body><w:p><w:t>主角设定</w:t></w:p></w:body></w:document>`))
	if err != nil {
		t.Fatal(err)
	}
	if !att.Extractable {
		t.Error(".docx 应可提取")
	}
	if !strings.Contains(att.Text, "主角设定") {
		t.Errorf("提取文本不符: %q", att.Text)
	}
	// 列表也应标记可提取
	list := s.List()
	if len(list) != 1 || !list[0].Extractable {
		t.Errorf("列表应标记 .docx 可提取: %+v", list)
	}
	// 聚合应包含 docx 文本
	agg := s.Aggregate()
	if !strings.Contains(agg, "主角设定") {
		t.Errorf("聚合应包含 docx 文本: %q", agg)
	}
}
