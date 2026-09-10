package service

import (
	"testing"

	"github.com/ReCasaOS/CasaOS-Common/utils/logger"
	"go.uber.org/goleak"
)

func TestSearch(t *testing.T) {
	logger.LogInitConsoleOnly()
	// CasaOS-Common/external pulls in orca-zhang/ecache, whose package init starts a
	// clock goroutine that never returns. It is running before this test does anything.
	goleak.VerifyNone(t, goleak.IgnoreCurrent())

	if d, e := NewOtherService().Search("test"); e != nil || d == nil {
		t.Error("then test search error", e)
	}
}
