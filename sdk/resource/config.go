// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"

	"github.com/karelbilek/opentelemetry/attribute"
)

type detectAttributes struct {
	attributes []attribute.KeyValue
}

func (d detectAttributes) Detect(context.Context) (*Resource, error) {
	return newWithAttributes(d.attributes...), nil
}
