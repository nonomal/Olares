/*
 Copyright 2021 The KubeSphere Authors.

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

package task

import (
	"github.com/beclab/Olares/cli/pkg/core/cache"
	"github.com/beclab/Olares/cli/pkg/core/connector"
	"github.com/beclab/Olares/cli/pkg/core/ending"
)

type Interface interface {
	GetName() string
	GetDesc() string
	Init(runtime connector.Runtime, moduleCache *cache.Cache, pipelineCache *cache.Cache)
	Execute() *ending.TaskResult
	ExecuteRollback()
}

type Tasks []Interface

func (t Tasks) InsertBefore(name string, task Interface) Tasks {
	if len(t) == 0 {
		return Tasks{task}
	}

	if name == "" {
		return append(Tasks{task}, t...)
	}

	for i, v := range t {
		if v.GetName() == name {
			return append(t[:i], append(Tasks{task}, t[i:]...)...)
		}
	}
	return append(t, task)
}

func (t Tasks) InsertAfter(name string, task Interface) Tasks {
	if len(t) == 0 {
		return Tasks{task}
	}

	if name == "" {
		return append(t, task)
	}

	for i, v := range t {
		if v.GetName() == name {
			return append(t[:i+1], append(Tasks{task}, t[i+1:]...)...)
		}
	}
	return append(t, task)
}
