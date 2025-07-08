<!--
Copyright (c) 2022 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# :lock: **Important Notice**
Starting with Container Storage Modules `v1.16.0`, this repository will transition to a closed-source model.<br>
* The current version remains open source and will continue to be available under the existing license.
* Customers will continue to receive access to enhanced features, timely updates, and official support through our commercial offerings.
* We remain committed to the open-source community - users engaging through Dell community channels will continue to receive guidance and support via Dell Support.

We sincerely appreciate the support and contributions from the community over the years.<br>
For access requests or inquiries, please contact the maintainers directly at [Dell Support](https://www.dell.com/support/kbdoc/en-in/000188046/container-storage-interface-csi-drivers-and-container-storage-modules-csm-how-to-get-support)

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

* [About](#about)
* [Code of Conduct](https://github.com/dell/csm/blob/main/docs/CODE_OF_CONDUCT.md)
* [Maintainer Guide](https://github.com/dell/csm/blob/main/docs/MAINTAINER_GUIDE.md)
* [Committer Guide](https://github.com/dell/csm/blob/main/docs/COMMITTER_GUIDE.md)
* [Contributing Guide](https://github.com/dell/csm/blob/main/docs/CONTRIBUTING.md)
* [List of Adopters](https://github.com/dell/csm/blob/main/docs/ADOPTERS.md)
* [Dell support](https://www.dell.com/support/incidents-online/en-us/contactus/product/container-storage-modules)
* [Security](https://github.com/dell/csm/blob/main/docs/SECURITY.md)
* [Certification Tool to Validate Drivers](#certification-tool-to-validate-drivers)

## About

_CERT-CSI_ is 100% open source and community-driven. All components are available
under [Apache 2 License](https://www.apache.org/licenses/LICENSE-2.0.html) on
GitHub.

## Certification Tool to Validate Drivers

Please refer [Cert-CSI Documentation](https://dell.github.io/csm-docs/docs/tooling/cert-csi/)
