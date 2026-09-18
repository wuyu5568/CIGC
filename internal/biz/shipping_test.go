package biz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cigc/app/internal/conf"
)

type memShipping struct {
	byUser map[uint64]*ShippingAddress
	next   uint64
}

func newMemShipping() *memShipping {
	return &memShipping{byUser: map[uint64]*ShippingAddress{}, next: 1}
}

func (m *memShipping) FindByUserID(_ context.Context, userID uint64) (*ShippingAddress, error) {
	a, ok := m.byUser[userID]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *a
	return &cp, nil
}

func (m *memShipping) Upsert(_ context.Context, a *ShippingAddress) (*ShippingAddress, error) {
	if a == nil || a.UserID == 0 {
		return nil, ErrShippingInvalid
	}
	now := time.Now()
	if old, ok := m.byUser[a.UserID]; ok {
		old.Name = a.Name
		old.Contact = a.Contact
		old.Address = a.Address
		old.UpdatedAt = now
		cp := *old
		return &cp, nil
	}
	cp := *a
	cp.ID = m.next
	m.next++
	cp.CreatedAt = now
	cp.UpdatedAt = now
	m.byUser[a.UserID] = &cp
	out := cp
	return &out, nil
}

func newShippingUC() (*UserUseCase, *memUsers, *memShipping) {
	users := newMemUsers()
	ship := newMemShipping()
	uc := NewUserUseCase(users, &memRecommends{path: map[uint64]string{}}, stubVerifier{}, stubTokens{}, &conf.Auth{}, "", nil)
	uc.SetShipping(ship)
	return uc, users, ship
}

func TestSaveShippingAddressRequiresFields(t *testing.T) {
	uc, users, _ := newShippingUC()
	u, err := users.Create(context.Background(), &User{Address: "0xabc"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.SaveShippingAddress(context.Background(), u.ID, "", "13800000000", "somewhere"); !errors.Is(err, ErrShippingInvalid) {
		t.Fatalf("empty name: %v", err)
	}
	if _, err := uc.SaveShippingAddress(context.Background(), u.ID, "张三", "  ", "somewhere"); !errors.Is(err, ErrShippingInvalid) {
		t.Fatalf("empty contact: %v", err)
	}
	if _, err := uc.SaveShippingAddress(context.Background(), u.ID, "张三", "13800000000", ""); !errors.Is(err, ErrShippingInvalid) {
		t.Fatalf("empty address: %v", err)
	}
}

func TestSaveShippingAddressUpsertsOnePerUser(t *testing.T) {
	uc, users, ship := newShippingUC()
	u, err := users.Create(context.Background(), &User{Address: "0xabc"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := uc.SaveShippingAddress(context.Background(), u.ID, " 张三 ", "13800000000", "北京路1号")
	if err != nil || got == nil || got.Name != "张三" || got.Contact != "13800000000" || got.Address != "北京路1号" {
		t.Fatalf("save %+v err=%v", got, err)
	}
	again, err := uc.SaveShippingAddress(context.Background(), u.ID, "李四", "13900000000", "上海路2号")
	if err != nil || again.ID != got.ID || again.Name != "李四" {
		t.Fatalf("upsert %+v err=%v", again, err)
	}
	if len(ship.byUser) != 1 {
		t.Fatalf("want 1 row, got %d", len(ship.byUser))
	}
	loaded, err := uc.GetShippingAddress(context.Background(), u.ID)
	if err != nil || loaded == nil || loaded.Name != "李四" || loaded.Contact != "13900000000" {
		t.Fatalf("get %+v err=%v", loaded, err)
	}
}

func TestRequireShippingAddress(t *testing.T) {
	uc, users, _ := newShippingUC()
	u, err := users.Create(context.Background(), &User{Address: "0xabc"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.RequireShippingAddress(context.Background(), u.ID); !errors.Is(err, ErrShippingRequired) {
		t.Fatalf("missing: %v", err)
	}
	if _, err := uc.SaveShippingAddress(context.Background(), u.ID, "张三", "13800000000", "北京路1号"); err != nil {
		t.Fatal(err)
	}
	got, err := uc.RequireShippingAddress(context.Background(), u.ID)
	if err != nil || !got.Complete() {
		t.Fatalf("require %+v err=%v", got, err)
	}
}

func TestGetShippingAddressNilRepo(t *testing.T) {
	uc := NewUserUseCase(newMemUsers(), &memRecommends{path: map[uint64]string{}}, stubVerifier{}, stubTokens{}, &conf.Auth{}, "", nil)
	got, err := uc.GetShippingAddress(context.Background(), 1)
	if err != nil || got != nil {
		t.Fatalf("nil repo: %+v err=%v", got, err)
	}
	if _, err := uc.RequireShippingAddress(context.Background(), 1); !errors.Is(err, ErrShippingRequired) {
		t.Fatalf("require nil repo: %v", err)
	}
}
