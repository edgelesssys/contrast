# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

from setuptools import setup

setup(
    name="igvm-signing-key",
    version="1.0.0",
    description="igvm-signing-key",
    long_description=open("README.md").read(),
    long_description_content_type="text/markdown",
    packages=['gen_signing_pem'],
    entry_points={
        'console_scripts': [
            'gen_signing_pem = gen_signing_pem.gen_signing_pem:main',
        ]},
    install_requires=[
        "ecdsa",
    ]
)
