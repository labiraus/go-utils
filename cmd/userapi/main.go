package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/labiraus/go-utils/pkg/api"
	"github.com/labiraus/go-utils/pkg/base"
	"github.com/labiraus/go-utils/pkg/kubernetesutil"
	"github.com/labiraus/go-utils/pkg/prometheusutil"

	"github.com/patrickmn/go-cache"
)

const (
	userHandlerLabel = "userHandler"
)

var (
	c          = cache.New(5*time.Minute, 10*time.Minute)
	kubeAccess = false
)

func main() {
	var err error
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
		if err != nil {
			slog.Error(err.Error())
		}
	}()
	ctx := base.Init("userapi")
	prometheusutil.Init()
	kubeAccess, err = kubernetesutil.Init()
	if err != nil {
		return
	}
	if !kubeAccess {
		slog.Info("kubernetes access not available")
	}

	http.HandleFunc("/user", userHandler)

	done := api.Init(ctx)
	close(base.Ready)
	<-done
	slog.Info("finishing")
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	startTime := time.Now() // Capture the start time
	prometheusutil.IncrementProcessed(userHandlerLabel, "call")
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
		if err != nil {
			slog.Error(err.Error())
			prometheusutil.IncrementProcessed(userHandlerLabel, "error")
		}
		prometheusutil.OpDuration(userHandlerLabel, time.Since(startTime))
	}()

	slog.Info("userHandler called")

	var request = UserRequest{}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if request.UserID == 0 {
		request.UserID = 1
	}

	secretValue, ok := c.Get("secretValue")
	if !ok {
		slog.Debug("reloading secret configValue")
		if !kubeAccess {
			secretValue = "no secret"
		} else {
			secretValue = base.GetEnv("SECRETVALUE", "no secret")
		}
		c.Set("secretValue", secretValue, cache.DefaultExpiration)
	}

	response := UserResponse{
		UserID:   request.UserID,
		Username: secretValue.(string),
		Email:    "something@somewhere.com",
	}

	data, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
