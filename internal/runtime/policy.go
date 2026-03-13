package runtime

import (
	"time"

	"github.com/jonlabelle/nssm-redux/internal/config"
)

const maxThrottleDelay = 4 * time.Minute

// RestartDecision describes the restart behavior after a process exits.
type RestartDecision struct {
	Action          config.ExitAction
	Delay           time.Duration
	ConsecutiveFast int
}

// DecideRestart computes the post-exit restart policy.
func DecideRestart(service config.Service, runtime time.Duration, consecutiveFast int, exitCode int) RestartDecision {
	action := service.ExitActionFor(exitCode)
	decision := RestartDecision{Action: action}
	if action != config.ExitActionRestart {
		return decision
	}

	if service.ThrottleDelay > 0 && runtime < service.ThrottleDelay {
		decision.ConsecutiveFast = consecutiveFast + 1
		backoff := time.Second << decision.ConsecutiveFast
		if backoff > maxThrottleDelay {
			backoff = maxThrottleDelay
		}
		decision.Delay = backoff
	} else {
		decision.ConsecutiveFast = 0
		decision.Delay = service.RestartDelay
	}

	if service.RestartDelay > decision.Delay {
		decision.Delay = service.RestartDelay
	}

	return decision
}
