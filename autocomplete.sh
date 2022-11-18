#!/usr/bin/env bash
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

