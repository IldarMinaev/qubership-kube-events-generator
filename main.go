package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	Logger    = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	count     = readEnv("COUNT", 5)
	sleep     = readEnv("SLEEP", 20)
	namespace = func(env string, defaultValue string) string {
		if value := os.Getenv(env); value != "" {
			return value
		}
		return defaultValue
	}("NAMESPACE", "logging")
	apiVKindName = func(env string, defaultValue []string) []string {
		result := strings.Split(os.Getenv(env), ",")
		if len(result) != 5 {
			Logger.Warn(fmt.Sprintf("Default value for %q is used", env), "value", defaultValue)
			return defaultValue
		}
		for _, res := range result {
			if res == "" {
				Logger.Warn(fmt.Sprintf("Default value for %q is used", env), "value", defaultValue)
				return defaultValue
			}
		}
		return result
	}("INVOLVEDOBJECT", []string{
		"integreatly.org/v1alpha1",
		"GrafanaDashboard",
		"graylog-grafana-dashboard-vm",
		"04e98ff7-7471-451f-a9cf-4bcad4a1bd41",
	})
)

func readEnv(env string, defaultValue int) int {
	value := os.Getenv(env)
	result, err := strconv.Atoi(value)
	if err != nil {
		Logger.Warn(fmt.Sprintf("Could not parse %q environment variable", env), "err", err)
	}
	if result > 0 {
		return result
	}
	Logger.Info(fmt.Sprintf("Using default value of %q environment variable", env), "value", defaultValue)
	return defaultValue
}

func main() {
	cfg := ctrl.GetConfigOrDie()
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		Logger.Error("Error building kubernetes client set", "err", err)
		os.Exit(1)
	}

	randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		createdCount := 0
		for i := 0; i < count; i++ {
			lastTs := metav1.Time{Time: time.Now()}
			eventName := fmt.Sprintf("test-%v-%v", i, randomizer.Intn(10000000))
			event := &corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventName,
					Namespace: namespace,
				},
				InvolvedObject: corev1.ObjectReference{
					Kind:            apiVKindName[1], //"GrafanaDashboard",
					Namespace:       namespace,
					Name:            apiVKindName[2],                       //"graylog-grafana-dashboard-vm",
					UID:             types.UID(apiVKindName[3]),            //"04e98ff7-7471-451f-a9cf-4bcad4a1bd41",
					APIVersion:      apiVKindName[0],                       //"integreatly.org/v1alpha1",
					ResourceVersion: strconv.Itoa(randomizer.Intn(100000)), //"78496715",
				},
				Message:        "This is test message of Event to load cloud-events-reader. Do not worry",
				FirstTimestamp: lastTs,
				LastTimestamp:  lastTs,
				Type:           "Normal",
				Reason:         "Completed",
				Source: corev1.EventSource{
					Component: "k8s-event-generator",
				},
				Count: int32(i),
			}
			if _, err = kubeClient.CoreV1().Events(namespace).Create(context.TODO(), event, metav1.CreateOptions{}); err != nil {
				Logger.Error("Could not create Event", "err", err)
			} else {
				createdCount++
				Logger.Info("Event created", "Event", eventName)
			}
		}
		Logger.Info(fmt.Sprintf("Sleeping for %v seconds. Created object count: %v", sleep, createdCount))
		time.Sleep(time.Duration(sleep) * time.Second)
	}
}
