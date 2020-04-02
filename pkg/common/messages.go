package common

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Messages struct {
	Info     []string
	Warnings []string
	Errors   []error
	Description string
}

func NewMessages(description string) *Messages {
	messages := Messages{}
	messages.Description = description
	messages.Info = []string{}
	messages.Warnings = []string{}
	messages.Errors = []error{}
	return &messages
}

func (m *Messages) LogInfo(message string, logger logr.Logger) {
	m.Info = append(m.Info, message)
	logger.Info(message)
}

func (m *Messages) LogWarning(message string, logger logr.Logger) {
	m.Warnings = append(m.Warnings, message)
	logger.V(2).Info(message)
}

func (m *Messages) LogError(err error, message string, logger logr.Logger) {
	m.Errors = append(m.Errors, err)
	m.Warnings = append(m.Warnings, message)
	logger.Info(message)
	logger.Error(err, message)
}

// Done when no error is informed and request is not set for requeue.
func (m *Messages) HandleComplete() reconcile.Result {
	return reconcile.Result{}
}
