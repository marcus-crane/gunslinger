package db

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/marcus-crane/gunslinger/models"
)

func TestCreate_GivesNoErrorForValidItem(t *testing.T) {
	t.Parallel()
	s := newMapStore()
	c := models.ComboDBMediaItem{
		ID:    1,
		Title: "Hi",
	}
	wantID := uint(1)
	gotID, err := s.Store(c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if wantID != gotID {
		t.Error(cmp.Diff(wantID, gotID))
	}
}
