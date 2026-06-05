package traffictesting_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/diff"
)

func newComparedRequest(t *testing.T, diffContent []diff.PatchOp) *traffictesting.Request {
	t.Helper()

	gateID := traffictesting.NewGateID()
	method, err := traffictesting.NewHTTPMethod("GET")
	require.NoError(t, err)
	path, err := traffictesting.ParsePath("/api/test")
	require.NoError(t, err)
	liveStatus, err := traffictesting.ParseStatusCode(200)
	require.NoError(t, err)
	shadowStatus, err := traffictesting.ParseStatusCode(500)
	require.NoError(t, err)

	createdAt := time.Now()
	live := traffictesting.Response{StatusCode: liveStatus, LatencyMs: 12, CreatedAt: createdAt}
	shadow := traffictesting.Response{StatusCode: shadowStatus, LatencyMs: 34, CreatedAt: createdAt}
	d, err := traffictesting.NewDiff(diffContent, traffictesting.DiffConfig{})
	require.NoError(t, err)

	req, err := traffictesting.NewRequest(
		gateID, method, path, "",
		traffictesting.NewHeaders(http.Header{}),
		nil, createdAt, live, shadow, *d,
	)
	require.NoError(t, err)
	return req
}

func TestRequest_RecordCompared_RaisesEvent(t *testing.T) {
	req := newComparedRequest(t, []diff.PatchOp{{}, {}, {}})

	require.Empty(t, req.PullEvents(), "no events before RecordCompared")

	req.RecordCompared()
	events := req.PullEvents()

	require.Len(t, events, 1)
	evt, ok := events[0].(traffictesting.RequestCompared)
	require.True(t, ok, "expected a RequestCompared event")

	assert.Equal(t, traffictesting.EventRequestCompared, evt.EventName())
	assert.Equal(t, req.GateID.String(), evt.GateID().String())
	assert.True(t, evt.HasDiff())
	assert.Len(t, evt.DiffContent(), 3)
	assert.Equal(t, 200, evt.LiveStatus())
	assert.Equal(t, 500, evt.ShadowStatus())
	assert.Equal(t, int64(12), evt.LiveLatencyMs())
	assert.Equal(t, int64(34), evt.ShadowLatencyMs())
	assert.Equal(t, req.CreatedAt, evt.OccurredAt())
}

func TestRequest_RecordCompared_NoDiff(t *testing.T) {
	req := newComparedRequest(t, []diff.PatchOp{})

	req.RecordCompared()
	events := req.PullEvents()

	require.Len(t, events, 1)
	evt := events[0].(traffictesting.RequestCompared)
	assert.False(t, evt.HasDiff(), "empty diff means responses matched")
}

func TestRequest_PullEvents_ClearsEvents(t *testing.T) {
	req := newComparedRequest(t, nil)

	req.RecordCompared()
	require.Len(t, req.PullEvents(), 1)
	assert.Empty(t, req.PullEvents(), "events are drained once pulled")
}
