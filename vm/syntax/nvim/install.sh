#!/bin/bash
ROOT=$(cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
HOME=$(eval echo ~$USER)
if [ -d "$HOME/.config/nvim" ]; then
  mkdir -p "$HOME/~/.config/nvim/{syntax,ftplugin}"
  cp $ROOT/syntax/*.vim "$HOME/.config/nvim/syntax/" && echo "copied $ROOT/syntax/*.vim to $HOME/.config/nvim/syntax/"
  cp $ROOT/ftplugin/*.vim "$HOME/.config/nvim/ftplugin/" && echo "copied $ROOT/ftplugin/*.vim to $HOME/.config/nvim/ftplugin/"
else
  echo "No .config/nvim directory found in $HOME. Skipping installation of neovim syntax files."
fi