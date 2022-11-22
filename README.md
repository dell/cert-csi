<!--
Copyright (c) 2022 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# CERT-CSI: Test tool for CSI Drivers

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v2.0%20adopted-ff69b4.svg)](docs/CODE_OF_CONDUCT.md) 
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)
[![Releases](https://img.shields.io/badge/Releases-green.svg)](https://github.com/dell/cert-csi/releases)

Test Certification Tool and Framework for Dell EMC CSI Drivers

This tool can be used for both functional and performance testing of any CSI Driver, using number of included test suites The metrics will be inserted in SQLite database in root of the project, and later can be converted in
easy-readable document format, where this metrics will be analyzed

> Since version 0.5.2 sqlite database created in the root of the project called like a storage class
>
>Ex: `--sc nfs` will generate `nfs.db` database file

## Table of Contents

* [Code of Conduct](./docs/CODE_OF_CONDUCT.md)
* [Maintainer Guide](./docs/MAINTAINER_GUIDE.md)
* [Committer Guide](./docs/COMMITTER_GUIDE.md)
* [Contributing Guide](./docs/CONTRIBUTING.md)
* [Branching Strategy](./docs/BRANCHING.md)
* [List of Adopters](./docs/ADOPTERS.md)
* [Maintainers](./docs/MAINTAINERS.md)
* [Support](./docs/SUPPORT.md)
* [Security](./docs/SECURITY.md)
* [About](#about)

## About

_CERT-CSI_ is 100% open source and community-driven. All components are available
under [Apache 2 License](https://www.apache.org/licenses/LICENSE-2.0.html) on
GitHub.
