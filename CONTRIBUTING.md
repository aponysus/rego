# Contributing to Recourse

Thank you for your interest in contributing to `recourse`! We welcome contributions from the community.

## Getting Started

For technical details on how to build, test, and understand the codebase, please read our **[Onboarding Guide](docs/onboarding.md)**.

## Code of Conduct

All contributors are expected to adhere to our [Code of Conduct](CODE_OF_CONDUCT.md). Please read it before participating.

## How to Contribute

1.  **Fork the repository** and create your branch from `main`.
2.  **Make sure tests pass**: Run `go test ./...`
3.  **Add tests**: If you add a feature or fix a bug, please add a regression test.
4.  **Format your code**: Run `gofmt -s -w .`
5.  **Submit a Pull Request**: Provide a clear description of the changes and link to any relevant issues.

## Style Guide

*   **Stdlib first**: We avoid third-party dependencies in the core `retry`, `policy`, and `controlplane` packages.
*   **Low cardinality**: Policy keys must not contain request-scoped data.
*   **Observability**: New features should emit events to the `observe` package.

## License

By contributing, you agree that your contributions will be licensed under the [Apache-2.0 License](LICENSE).
