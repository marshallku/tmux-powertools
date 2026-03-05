BINARY_NAME := tmux-powertools
INSTALL_DIR := $(HOME)/.local/bin

.PHONY: build install uninstall clean

build:
	go build -o $(BINARY_NAME) .

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME)
