package pipe

import (
	"github.com/mariomac/go-pipes/pkg/pipe/internal/refl"
)

// todo: set as a builderRunner configurable property
const channelsBuf = 20

// the connector is the output channel of the previous stage (nil for the first stage),
// that is used as input for the next stage.
func (b *builderRunner) run(connector *refl.Channel) {
	for _, stg := range b.line {
		if stg.fork != nil {
			left, right := connector.Fork()
			stg.fork.left.run(&left)
			stg.fork.right.run(&right)
			// end of this line, not invoking more items
			return
		}
		invoke(stg.function, connector)
	}
}

// the connector is passed as argument to the function to be run. If the function returns a
// channel (first or middle stages), the connector is updated to it, so it will be passed to the
// next stage
func invoke(fn refl.Function, connector *refl.Channel) {
	if connector.IsNil() {
		// output-only function (first element of pipeline)
		*connector = fn.RunAsStartGoroutine(channelsBuf)
	} else if fn.NumArgs() == 1 {
		// input-only function (last element of pipeline)
		fn.RunAsEndGoroutine(*connector)
	} else {
		// intermediate stage of the pipeline with input and output channel
		*connector = fn.RunAsMiddleGoroutine(*connector, channelsBuf)
	}
}
