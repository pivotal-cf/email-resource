#!/bin/bash -e
cp source/Dockerfile output/.
mkdir -p output/bin
cp releases/in output/bin/.
cp releases/out output/bin/.
cp releases/check output/bin/.
chmod +x output/bin/*
cp releases/version output/.
