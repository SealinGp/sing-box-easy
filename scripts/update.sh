#!/bin/bash
set -e
ver=$1
wget https://github.com/SealinGp/sing-box-easy/releases/download/$ver/sing-box-easy-linux-amd64.tar.gz
tar -xvzf sing-box-easy-linux-amd64.tar.gz
rm sing-box-easy-linux-amd64.tar.gz