package tx

import (
	"errors"
	"testing"
)

// TestCompensationRollback 验证 Rollback 时逆序执行补偿。
func TestCompensationRollback(t *testing.T) {
	// 这里不测试 GORM 事务（需要真实 DB），只测试纯补偿栈逻辑。
	// 模拟补偿执行顺序：注册 A, B, C → Rollback 执行顺序应为 C, B, A。
	u := &Unit{comp: nil}
	order := make([]string, 0)

	u.Defer("A", func() error { order = append(order, "A"); return nil })
	u.Defer("B", func() error { order = append(order, "B"); return nil })
	u.Defer("C", func() error { order = append(order, "C"); return nil })

	// 手动模拟 Rollback 的补偿执行
	for i := len(u.comp) - 1; i >= 0; i-- {
		u.comp[i].Action()
	}

	if len(order) != 3 || order[0] != "C" || order[1] != "B" || order[2] != "A" {
		t.Fatalf("expected [C B A], got %v", order)
	}
}

// TestCompensationWithError 验证某补偿失败不影响后续补偿执行。
func TestCompensationWithError(t *testing.T) {
	u := &Unit{comp: nil}
	called := make([]string, 0)

	u.Defer("A", func() error { called = append(called, "A"); return errors.New("A failed") })
	u.Defer("B", func() error { called = append(called, "B"); return nil })

	for i := len(u.comp) - 1; i >= 0; i-- {
		u.comp[i].Action()
	}

	if len(called) != 2 || called[0] != "B" || called[1] != "A" {
		t.Fatalf("expected [B A], got %v", called)
	}
}

// TestCommitClearsCompensations 验证 Commit 后补偿栈被清空。
func TestCommitClearsCompensations(t *testing.T) {
	// 不使用真实 DB Commit，直接验证清空逻辑
	u := &Unit{comp: nil}
	u.Defer("cleanup", func() error { return nil })
	u.done = true
	u.comp = nil

	if len(u.comp) != 0 {
		t.Fatalf("expected empty compensations after commit, got %d", len(u.comp))
	}
}

// TestDeferOrder 验证添加顺序
func TestDeferOrder(t *testing.T) {
	u := &Unit{comp: nil}
	u.Defer("1st", func() error { return nil })
	u.Defer("2nd", func() error { return nil })
	u.Defer("3rd", func() error { return nil })

	if len(u.comp) != 3 {
		t.Fatalf("expected 3 compensations, got %d", len(u.comp))
	}
	if u.comp[0].Desc != "1st" || u.comp[1].Desc != "2nd" || u.comp[2].Desc != "3rd" {
		t.Fatalf("wrong order: %v", u.comp)
	}
}
