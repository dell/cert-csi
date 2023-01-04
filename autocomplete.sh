#!/usr/bin/env bash
#
#
# Copyright © 2022-2023 Dell Inc. or its subsidiaries. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#      http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
#

echo "$(whoami)"

if cat /etc/shells | grep "zsh"
then
  echo "zsh shell detected"
	cp -f ./zsh_autocomplete ~/.cert-zsh-autocomplete
	echo """PROG=cert-csi
_CLI_ZSH_AUTOCOMPLETE_HACK=1
source  ~/.cert-zsh-autocomplete
""" >> ~/.zshrc
fi
if cat /etc/shells | grep "bash"
then
  echo "bash shell detected"
	cp -f ./bash_autocomplete /etc/bash_completion.d/cert-csi
	source /etc/bash_completion.d/cert-csi
else
  echo "Was unable to detect any suitable shells. We don't use fish here buddy"
fi

[ "$UID" -eq 0 ] || exec sudo "$0" "$@"

cp -f ./cert-csi /usr/local/bin/

