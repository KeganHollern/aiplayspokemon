NAME := dawngb

ifeq ($(OS),Windows_NT)
EXE := .exe
else
EXE :=
endif

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
GODOT?=$(shell which godot)

.PHONY: build wasm goenv profile godot libretro clean

build:
	go build -o build/$(NAME)$(EXE) ./src/ebi

run:
	go run ./src/ebi ./rom/PokemonYellow.gb

clean:
	rm -rf build
