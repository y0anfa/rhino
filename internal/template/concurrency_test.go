package template

import (
	"fmt"
	"sync"
	"testing"

	"github.com/y0anfa/rhino/internal/providers"
)

// Workflow.Run shares one Context across tasks that execute in parallel: results
// are recorded while sibling tasks are still resolving their params.
func TestContext_ConcurrentUse(t *testing.T) {
	ctx := NewContext("wf", "desc", "trigger", "manual")

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			ctx.SetTaskResult(fmt.Sprintf("task%d", i), &providers.TaskResult{
				Output:   "out",
				Metadata: map[string]string{"k": "v"},
			})
		}(i)
		go func(i int) {
			defer wg.Done()
			ResolveParams(map[string]interface{}{
				"a": fmt.Sprintf("{{task.task%d.output}}", i),
				"b": fmt.Sprintf("{{task.task%d.metadata.k}}", i),
			}, ctx)
		}(i)
	}
	wg.Wait()
}
