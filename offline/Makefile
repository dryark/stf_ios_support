all: libimd

# --- Basic Directories ---

repos:
	mkdir repos

# --- LibIMobileDevice ---

libimd: /usr/local/bin/ideviceinfo

/usr/local/bin/ideviceinfo: repos/libimobiledevice repos/libimobiledevice/tools/ideviceinfo | repos/libimobiledevice
	$(MAKE) -C repos/libimobiledevice install

repos/libimobiledevice/tools/ideviceinfo: repos/libimobiledevice repos/libimobiledevice/Makefile | repos/libimobiledevice
	$(MAKE) -C repos/libimobiledevice

repos/libimobiledevice/Makefile: | repos/libimobiledevice
	cd repos/libimobiledevice && NOCONFIGURE=1 ./autogen.sh
	cd repos/libimobiledevice && ./configure --disable-openssl

# --- Clones ---

repos/libimobiledevice: | repos
	git clone https://github.com/libimobiledevice/libimobiledevice.git repos/libimobiledevice