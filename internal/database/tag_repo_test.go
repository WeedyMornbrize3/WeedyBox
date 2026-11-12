// internal/database/tag_repo_test.go
//
// 用户自定义标签的仓储测试：表惰性创建、先读不报错、重复添加幂等、来源隔离、删除幂等。
// 与 library_repo_test 同一套临时库辅助函数（newTestService / tableExists）。
package database

import (
	"context"
	"strings"
	"testing"
)

func TestWorkTagsLazyTableAndCRUD(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	// ① 表还不存在时先读/先删：返回空、不报错，也不该把表建出来
	tags, err := s.ListWorkTags(ctx, SourceDLsite, "rj01721474")
	if err != nil {
		t.Fatalf("空表读取失败: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("空表应返回空标签，实际 %v", tags)
	}

	// ② 添加：编号归一化 + 重复添加幂等
	if err := s.AddWorkTag(ctx, SourceDLsite, "rj01721474", "想玩"); err != nil {
		t.Fatalf("添加标签失败: %v", err)
	}
	if err := s.AddWorkTag(ctx, SourceDLsite, "RJ01721474", "想玩"); err != nil {
		t.Fatalf("重复添加应幂等，实际报错: %v", err)
	}
	if !tableExists(t, s, "work_tags") {
		t.Fatal("写入后应创建 work_tags 表")
	}
	tags, err = s.ListWorkTags(ctx, SourceDLsite, "RJ01721474")
	if err != nil {
		t.Fatalf("查询标签失败: %v", err)
	}
	if got := strings.Join(tags, ","); got != "想玩" {
		t.Errorf("标签 = %q，期望「想玩」（重复添加不应出现两次）", got)
	}

	// ③ 顺序按添加先后
	if err := s.AddWorkTag(ctx, SourceDLsite, "RJ01721474", "已通关"); err != nil {
		t.Fatalf("追加标签失败: %v", err)
	}
	tags, _ = s.ListWorkTags(ctx, SourceDLsite, "RJ01721474")
	if got := strings.Join(tags, ","); got != "想玩,已通关" {
		t.Errorf("标签顺序 = %q，期望「想玩,已通关」", got)
	}

	// ④ 来源隔离：别的来源下同名编号互不影响
	if err := s.AddWorkTag(ctx, "steam", "RJ01721474", "Steam 标签"); err != nil {
		t.Fatalf("其它来源添加失败: %v", err)
	}
	tags, _ = s.ListWorkTags(ctx, SourceDLsite, "RJ01721474")
	if len(tags) != 2 {
		t.Errorf("dlsite 来源标签数 = %d，期望 2（不应混入别的来源）", len(tags))
	}
	other, _ := s.ListWorkTags(ctx, "steam", "RJ01721474")
	if len(other) != 1 || other[0] != "Steam 标签" {
		t.Errorf("steam 来源标签 = %v", other)
	}

	// ⑤ 删除：删一个少一个；删不存在的也成功
	if err := s.RemoveWorkTag(ctx, SourceDLsite, "RJ01721474", "想玩"); err != nil {
		t.Fatalf("删除标签失败: %v", err)
	}
	if err := s.RemoveWorkTag(ctx, SourceDLsite, "RJ01721474", "不存在的标签"); err != nil {
		t.Errorf("删除不存在的标签应幂等成功，实际 %v", err)
	}
	tags, _ = s.ListWorkTags(ctx, SourceDLsite, "RJ01721474")
	if len(tags) != 1 || tags[0] != "已通关" {
		t.Errorf("删除后标签 = %v，期望只剩「已通关」", tags)
	}
}
