# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

from ecdsa import SigningKey, NIST384p
from hashlib import sha384

def main():
    print(SigningKey.generate(curve=NIST384p, hashfunc=sha384).to_pem().decode('utf-8'))
