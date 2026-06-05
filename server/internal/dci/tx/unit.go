// Package tx provides a lightweight UnitOfWork with Saga-style compensation,
// wrapping GORM's database transaction.
//
// Usage:
//
//	u := tx.NewUnit(db)
//	u.Begin()
//	u.Defer("cleanup-file", func() error { return os.Remove(path) })
//	if err := doWork(u.DB()); err != nil {
//	    return u.Rollback() // runs compensations + DB rollback
//	}
//	return u.Commit()
package tx

import (
	"fmt"
	"sync"

	"gorm.io/gorm"
)

// Comp 是单个补偿动作的描述。
type Comp struct {
	Desc   string
	Action func() error
}

// Unit 封装一个 GORM 数据库事务和一个补偿栈。
// 补偿只在 Rollback 时逆序执行；Commit 后补偿栈被清空。
type Unit struct {
	db   *gorm.DB
	mu   sync.Mutex
	comp []Comp
	done bool
}

// NewUnit 创建一个未开始事务的 Unit。
func NewUnit(db *gorm.DB) *Unit {
	return &Unit{db: db}
}

// Begin 开启 GORM 事务。
func (u *Unit) Begin() error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.done {
		return fmt.Errorf("tx: unit already finalized")
	}
	u.db = u.db.Begin()
	if u.db.Error != nil {
		return fmt.Errorf("tx: begin: %w", u.db.Error)
	}
	return nil
}

// DB 返回当前事务内的 *gorm.DB。
// 调用前必须先 Begin()。
func (u *Unit) DB() *gorm.DB {
	return u.db
}

// RawDB 返回底层的非事务 *gorm.DB（用于只读操作）。
func (u *Unit) RawDB() *gorm.DB {
	return u.db
}

// Defer 注册一个补偿函数。事务提交成功后补偿不会被调用；
// Rollback 时按注册的逆序执行所有补偿。
func (u *Unit) Defer(desc string, fn func() error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.comp = append(u.comp, Comp{Desc: desc, Action: fn})
}

// Commit 提交 DB 事务并清空补偿栈。
func (u *Unit) Commit() error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.done {
		return fmt.Errorf("tx: unit already finalized")
	}
	u.done = true
	if err := u.db.Commit().Error; err != nil {
		return fmt.Errorf("tx: commit: %w", err)
	}
	u.comp = nil
	return nil
}

// Rollback 回滚 DB 事务，逆序执行所有补偿函数。
func (u *Unit) Rollback() error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.done {
		return fmt.Errorf("tx: unit already finalized")
	}
	u.done = true

	rbErr := u.db.Rollback().Error

	// 逆序执行补偿
	var compErr error
	for i := len(u.comp) - 1; i >= 0; i-- {
		if err := u.comp[i].Action(); err != nil {
			if compErr == nil {
				compErr = err
			}
		}
	}

	if rbErr != nil {
		return fmt.Errorf("tx: rollback: %w", rbErr)
	}
	if compErr != nil {
		return fmt.Errorf("tx: compensation: %w", compErr)
	}
	return nil
}
