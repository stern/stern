//   Copyright 2016 Wercker Holding BV
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package main

import (
	"log"
	"os"

	"github.com/henriknelson/stern/cmd"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func main() {
	streams := genericclioptions.IOStreams{Out: os.Stdout, ErrOut: os.Stderr}
	stern, err := cmd.NewSternCmd(streams)
	if err != nil {
		log.Fatal(err)
	}

	if err := stern.Execute(); err != nil {
		os.Exit(1)
	}
}
