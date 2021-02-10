/*
Copyright the Cluster API Provider BYOH contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cloudinit

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

type unsupportedAction struct {
	module string
}

func newUnsupportedAction(module string) action {
	return &unsupportedAction{module: module}
}

func (u *unsupportedAction) Unmarshal(data []byte) error {
	return nil
}

func (u *unsupportedAction) Run(_ context.Context, _ logr.Logger) error {
	return errors.Errorf("cloud init adapter:: unsupported command %q", u.module)
}
