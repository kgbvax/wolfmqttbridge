#!/bin/sh
node /authcode-resolver/resolve-authcode.js &
sleep 5
/wolfmqttbridge
