# Testkube Licensing FAQ

Testkube's software licensing is designed to be transparent and to support both open source and commercial use cases. This document aims to address common questions related to our licensing model.

## Licenses

Testkube software is distributed under two primary licenses:
- **MIT License (MIT)**: A permissive open-source license that allows for broad freedom in usage and modification.
- **Testkube Community License (TCL)**: A custom license designed to protect the Testkube community and ecosystem, covering specific advanced features.

## Testkube Core

Testkube Core is free to use. Most core features are licensed under the MIT license, but some core features are subject to the TCL.

## Testkube Pro

Testkube Pro features require a paid license from Testkube (see [pricing](https://testkube.io/pricing)) and are licensed under the Testkube Community License.

:::note
You can find any feature's license by checking the code's file header in the Testkube repository.
:::

### What is the TCL License?

The Testkube Community License (TCL) is a custom license created by Testkube to cover certain aspects of the Testkube software. It was inspired by the [CockroachDB Community License](https://www.cockroachlabs.com/docs/stable/licensing-faqs#ccl) and designed to ensure that advanced features and proprietary extensions remain available and maintained for the community while allowing Testkube to sustain its development through commercial offerings.

### Why does Testkube have a dual-licensing scheme with MIT / TCL?

Testkube uses a dual license model to balance open source community participation with the ability to fund continued development. Core functionality is available under the permissive MIT license, while advanced features require a commercial license. This allows the community to benefit from an open source project while providing a sustainability model.

### How does the TCL license apply to Testkube Core?

Testkube core functionality is available under the MIT license, allowing free usage, modification and distribution. However, advanced pro features are covered under the more restrictive TCL. Contributions back to Testkube Core are welcomed, but modifications to TCL-licensed components may require reaching out to Testkube first.

### Can I use Testkube Core for free?

Yes, Testkube Core can be used for free. The majority of Testkube's core functionalities are available under the MIT license, which allows for free usage, modification, and distribution.

### Does the TCL license restrict my usage of Testkube Core?

No, the TCL license only applies to specific advanced features marked as "Pro" in the codebase. It does not restrict usage of the MIT-licensed open source components.

### Can I make changes to Testkube Core for my own usage?

Yes, you are free to make changes to Testkube Core components licensed under the MIT license for your own use. For components under the TCL, you must adhere to the terms of that license, which include restrictions on redistribution or commercial use, for this we advise you to reach out to us first.

### Can I make contributions back to Testkube Core?

Yes! Contributions are welcomed, whether bug fixes, enhancements or documentation. As long as you retain the existing MIT license, contributions can be made freely.

## Feature Licensing

The table below shows how certain core and pro features in the GitHub repository are licensed:

| Feature          | Core/MIT    | Pro/TCL  |
| :---             |    :----:   |    :---: |
| Tests            |      x      |          |
| Basic Testsuites |      x      |          |
| Triggers         |      x      |          |
| Executors        |      x      |          |
| Webhooks         |      x      |          |
| Sources          |      x      |          |
| Test Workflows   |             |    x     |
| Adv Testsuites   |             |    x     |

