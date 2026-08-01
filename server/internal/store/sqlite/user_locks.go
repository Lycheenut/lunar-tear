package sqlite

import (
	"sort"
	"sync"
)

type userLockRef struct {
	mu   sync.Mutex
	refs int
}

type userLockManager struct {
	mu     sync.Mutex
	byUser map[int64]*userLockRef
}

// lock acquires distinct user locks in ascending id order. The fixed order
// prevents two multi-user updates from deadlocking when callers supply the
// same ids in different orders.
func (m *userLockManager) lock(userIds []int64) ([]int64, func()) {
	ids := append([]int64(nil), userIds...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	unique := ids[:0]
	for _, id := range ids {
		if len(unique) == 0 || unique[len(unique)-1] != id {
			unique = append(unique, id)
		}
	}

	m.mu.Lock()
	if m.byUser == nil {
		m.byUser = make(map[int64]*userLockRef)
	}
	locks := make([]*userLockRef, len(unique))
	for i, id := range unique {
		ref := m.byUser[id]
		if ref == nil {
			ref = &userLockRef{}
			m.byUser[id] = ref
		}
		ref.refs++
		locks[i] = ref
	}
	m.mu.Unlock()

	for _, ref := range locks {
		ref.mu.Lock()
	}
	return unique, func() {
		for i := len(locks) - 1; i >= 0; i-- {
			locks[i].mu.Unlock()
		}
		m.mu.Lock()
		for i, id := range unique {
			locks[i].refs--
			if locks[i].refs == 0 {
				delete(m.byUser, id)
			}
		}
		m.mu.Unlock()
	}
}
