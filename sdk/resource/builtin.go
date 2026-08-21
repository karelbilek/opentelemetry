// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/karelbilek/opentelemetry/attribute"
	"github.com/karelbilek/opentelemetry/sdk"
	semconv "github.com/karelbilek/opentelemetry/semconv"
)

type (
	// telemetrySDK is a Detector that provides information about
	// the OpenTelemetry SDK used.  This Detector is included as a
	// builtin. If these resource attributes are not wanted, use
	// resource.New() to explicitly disable them.
	telemetrySDK struct{}

	// host is a Detector that provides information about the host
	// being run on. This Detector is included as a builtin. If
	// these resource attributes are not wanted, use the
	// resource.New() to explicitly disable them.
	host struct{}

	stringDetector struct {
		K attribute.Key
		F func() (string, error)
	}

	defaultServiceNameDetector struct{}

	defaultServiceInstanceIDDetector struct{}

	fixedServiceNameDetector struct{ s string }
)

var (
	_ Detector = telemetrySDK{}
	_ Detector = host{}
	_ Detector = stringDetector{}
	_ Detector = defaultServiceNameDetector{}
	_ Detector = defaultServiceInstanceIDDetector{}
	_ Detector = fixedServiceNameDetector{}
)

// Detect returns a *Resource that describes the OpenTelemetry SDK used.
func (telemetrySDK) Detect(context.Context) (*Resource, error) {
	return newWithAttributes(
		semconv.TelemetrySDKName("github.com/karelbilek/opentelemetry"),
		semconv.TelemetrySDKLanguageGo,
		semconv.TelemetrySDKVersion(sdk.Version()),
	), nil
}

// Detect returns a *Resource that describes the host being run on.
func (host) Detect(ctx context.Context) (*Resource, error) {
	return StringDetector(semconv.HostNameKey, os.Hostname).Detect(ctx)
}

// StringDetector returns a Detector that will produce a *Resource
// containing the string as a value corresponding to k.
func StringDetector(k attribute.Key, f func() (string, error)) Detector {
	return stringDetector{K: k, F: f}
}

// Detect returns a *Resource that describes the string as a value
// corresponding to attribute.Key
func (sd stringDetector) Detect(context.Context) (*Resource, error) {
	value, err := sd.F()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", string(sd.K), err)
	}
	a := sd.K.String(value)
	if !a.Valid() {
		return nil, fmt.Errorf("invalid attribute: %q -> %q", a.Key, a.Value.String())
	}
	return newWithAttributes(sd.K.String(value)), nil
}

// Detect implements Detector.
func (defaultServiceNameDetector) Detect(ctx context.Context) (*Resource, error) {
	return StringDetector(
		semconv.ServiceNameKey,
		func() (string, error) {
			executable, err := os.Executable()
			if err != nil {
				return "unknown_service:go", nil
			}
			return "unknown_service:" + filepath.Base(executable), nil
		},
	).Detect(ctx)
}

func (f fixedServiceNameDetector) Detect(ctx context.Context) (*Resource, error) {
	return StringDetector(
		semconv.ServiceNameKey,
		func() (string, error) {
			return f.s, nil
		},
	).Detect(ctx)
}

var defaultUUID = makeUUID()

func makeUUID() string {
	var u [16]byte
	_, _ = rand.Read(u[:])
	u[6] = (u[6] & 0x0f) | 0x40 // version 4
	u[8] = (u[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}

// Detect implements Detector.
func (defaultServiceInstanceIDDetector) Detect(ctx context.Context) (*Resource, error) {
	return StringDetector(
		semconv.ServiceInstanceIDKey,
		func() (string, error) {
			return defaultUUID, nil
		},
	).Detect(ctx)
}
