// internal/database/tag_repo_test.go
//
// 作品标签的仓储测试：表惰性创建、先读不报错、整份覆盖保序、"已改写"标记、
// 空列表也算改写、来源隔离、编号大小写归一化。
// 与 library_repo_test 同一套临时库辅助函数（newTestService / tableExists）。
package database

import (
	"context"
	"strings"
	"testing"
)

func TestWorkTagsSaveAndEditFlag(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	// ① 表还不存在时先读：返回空、不报错，"已改写"为 false
	tags, err := s.ListWorkTags(ctx, SourceDLsite, "rj01721474")
	if err != nil {
		t.Fatalf("空表读取失败: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("空表应返回空标签，实际 %v", tags)
	}
	edited, err := s.WorkTagsEdited(ctx, SourceDLsite, "rj01721474")
	if err != nil {
		t.Fatalf("读标记失败: %v", err)
	}
	if edited {
		t.Error("还没保存过，不该是已改写")
	}

	// ② 整份保存：顺序按传入列表（不是字母序），编号归一化
	if err := s.SaveWorkTags(ctx, SourceDLsite, "rj01721474", []string{"想玩", "已通关", "神作"}); err != nil {
		t.Fatalf("保存标签失败: %v", err)
	}
	if !tableExists(t, s, "work_tags") || !tableExists(t, s, "work_tag_edits") {
		t.Fatal("保存后应创建 work_tags 与 work_tag_edits 两张表")
	}
	tags, err = s.ListWorkTags(ctx, SourceDLsite, "RJ01721474")
	if err != nil {
		t.Fatalf("查询标签失败: %v", err)
	}
	if got := strings.Join(tags, ","); got != "想玩,已通关,神作" {
		t.Errorf("标签 = %q，期望「想玩,已通关,神作」", got)
	}
	edited, _ = s.WorkTagsEdited(ctx, SourceDLsite, "RJ01721474")
	if !edited {
		t.Error("保存过后应为已改写")
	}

	// ③ 再次保存是覆盖而不是追加：删掉的不该回来
	if err := s.SaveWorkTags(ctx, SourceDLsite, "RJ01721474", []string{"已通关"}); err != nil {
		t.Fatalf("覆盖保存失败: %v", err)
	}
	tags, _ = s.ListWorkTags(ctx, SourceDLsite, "RJ01721474")
	if got := strings.Join(tags, ","); got != "已通关" {
		t.Errorf("覆盖后标签 = %q，期望只剩「已通关」", got)
	}

	// ④ 保存空列表 = 用户把标签全删光：列表变空，但"已改写"必须保持 true
	if err := s.SaveWorkTags(ctx, SourceDLsite, "RJ01721474", nil); err != nil {
		t.Fatalf("保存空列表失败: %v", err)
	}
	tags, _ = s.ListWorkTags(ctx, SourceDLsite, "RJ01721474")
	if len(tags) != 0 {
		t.Errorf("保存空列表后应没有标签，实际 %v", tags)
	}
	edited, _ = s.WorkTagsEdited(ctx, SourceDLsite, "RJ01721474")
	if !edited {
		t.Error("保存空列表也应是已改写（否则前端会把抓取标签又并回来）")
	}

	// ⑤ 来源隔离：同一编号在别的来源下互不影响
	if err := s.SaveWorkTags(ctx, "steam", "RJ01721474", []string{"Steam 标签"}); err != nil {
		t.Fatalf("其它来源保存失败: %v", err)
	}
	tags, _ = s.ListWorkTags(ctx, SourceDLsite, "RJ01721474")
	if len(tags) != 0 {
		t.Errorf("dlsite 来源标签应仍为空，实际 %v", tags)
	}
	other, _ := s.ListWorkTags(ctx, "steam", "RJ01721474")
	if len(other) != 1 || other[0] != "Steam 标签" {
		t.Errorf("steam 来源标签 = %v", other)
	}
}
