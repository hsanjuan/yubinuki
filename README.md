YubiNuki
========

This program allows to use NFC-enabled Yubikeys to open Nuki locks.

Requirements
------------

You will need:

* A libnfc-compatible NFC reader and `libnfc-devel` (or equivalent) installed and configured (see `/etc/nfc/libnfc.conf`) in your system (i.e. the [GO2NFC141U from Elechouse](https://www.elechouse.com/elechouse/index.php?main_page=product_info&cPath=90_93&products_id=2253&zenid=ei93tidcjbuo4aj6inm1ahq163)
* A [Nuki Bridge and Lock](https://nuki.io/de/shop/) and configured API access to it (https://developer.nuki.io/t/bridge-http-api/26)
* An [NFC Yubikey](https://www.yubico.com/store/) (currently only tested with the Yubikey NEO)
* A Yubicloud user ID and Secret (https://upgrade.yubico.com/getapikey/)
* [`Go` compiler](https://golang.org/dl/)

This program runs fine on a Raspberry Pi Zero W.

Installation
------------

After installing `Go`, in order to run `yubinuki` in your system:

* Download and compile the `yubinuki` program
* Copy the executable to `/usr/local/bin`
* Copy the configuration file to `/etc/yubinuki.json` and set it up
* Install and `yubinuki.service` file and enable the service

In other words, from the repository folder, run:

```sh
go get github.com/hsanjuan/yubinuki
cd $GOPATH/src/github.com/hsanjuan/yubinuki
cp yubinuki.template.json yubinuki.json
```

At this point, EDIT `yubinuki.json` with the right configuration values. Then:

```sh
sudo cp $GOPATH/bin/yubinuki /usr/local/bin/yubinuki
sudo mv yubinuki.json /etc/yubinuki.json
sudo chown root:root /etc/yubinuki.json
sudo chmod 0600 /etc/yubinuki.json
sudo cp yubinuki.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable yubinuki
sudo systemctl start yubinuki
```
