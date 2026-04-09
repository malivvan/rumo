#!/bin/bash
ROOT=$(cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
HOME=$(eval echo ~$USER)
if [ -d "$HOME/.vim" ]; then
  mkdir -p "$HOME/.vim/{syntax,ftdetect}"
  cp $ROOT/syntax/*.vim "$HOME/.vim/syntax/" && echo "copied $ROOT/syntax/*.vim to $HOME/.vim/syntax/"
  cp $ROOT/ftdetect/*.vim "$HOME/.vim/ftdetect/" && echo "copied $ROOT/ftdetect/*.vim to $HOME/.vim/ftdetect/"
else
  echo "No .vim directory found in $HOME. Skipping installation of vim syntax files."
fi