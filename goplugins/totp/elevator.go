package totp

import (
	"fmt"
	"sync"
	"time"

	otp "github.com/dgryski/dgoogauth"
	"github.com/uva-its/gopherbot/bot"
)

var timeoutLock sync.RWMutex
var lastElevate map[string]time.Time

type timeoutType int

const (
	idle timeoutType = iota
	absolute
)

type config struct {
	TimeoutSeconds int
	tf64           float64
	TimeoutType    string
	tt             timeoutType
}

var cfg config

func checkOTP(r *bot.Robot, code string) (bool, bot.RetVal) {
	var userOTP otp.OTPConfig
	lock, exists, ret := r.CheckoutDatum(r.User, &userOTP, true)
	if ret != bot.Ok {
		r.CheckinDatum(r.User, lock)
		return false, bot.GeneralError
	}
	if !exists {
		r.CheckinDatum(r.User, lock)
		return false, ret
	}
	valid, err := userOTP.Authenticate(code)
	if err != nil {
		r.Log(bot.Error, fmt.Errorf("Problem authenticating launch code for user %s: %v", r.User, err))
		r.CheckinDatum(r.User, lock)
		return false, bot.TechnicalProblem
	}
	ret = r.UpdateDatum(r.User, lock, &userOTP)
	if ret != bot.Ok {
		r.Log(bot.Error, fmt.Errorf("Problem updating OTP for %s, failing", r.User))
		return false, ret
	}
	return valid, bot.Ok
}

func getcode(r *bot.Robot, immediate bool) bool {
	dm := ""
	if r.Channel != "" {
		dm = " - I'll message you directly"
	}
	if immediate {
		r.Say("This command requires immediate elevation" + dm)
	} else {
		r.Say("This command requires elevation" + dm)
	}
	r.Pause(1)
	r.Direct().Say("Please provide your totp launch code")
	rep, ret := r.Direct().WaitForReply("OTP", 30)
	if ret != bot.Ok {
		r.Direct().Say("Try again? I need a 6-digit launch code")
		rep, ret = r.Direct().WaitForReply("OTP", 30)
	}
	if ret == bot.Ok {
		ok, ret := checkOTP(r, rep)
		if ret != bot.Ok {
			r.Direct().Say("There were technical issues validating your code, ask an administrator to check the log")
			return false
		}
		return ok
	}
	return false
}

func elevate(r *bot.Robot, immediate bool) bool {
	allowed := false
	now := time.Now().UTC()
	if immediate {
		allowed = getcode(r, immediate)
	} else {
		timeoutLock.RLock()
		le, ok := lastElevate[r.User]
		timeoutLock.RUnlock()
		ask := false
		if ok {
			diff := now.Sub(le)
			if diff.Seconds() > cfg.tf64 {
				ask = true
			} else {
				allowed = true
			}
		} else {
			ask = true
		}
		if ask {
			allowed = getcode(r, immediate)
		}
	}
	if allowed && cfg.tt == idle {
		timeoutLock.Lock()
		lastElevate[r.User] = now
		timeoutLock.Unlock()
	}
	return allowed
}

func provider(r bot.Handler) bot.Elevate {
	botHandler = r
	botHandler.GetElevateConfig(&cfg)
	if cfg.TimeoutType == "absolute" {
		cfg.tt = absolute
	}
	cfg.tf64 = float64(cfg.TimeoutSeconds)
	return elevate
}

func init() {
	bot.RegisterElevator("totp", provider)
	lastElevate = make(map[string]time.Time)
}
