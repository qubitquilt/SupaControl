# Documentation Guide

This guide provides instructions and best practices for writing and contributing to the SupaControl documentation.

## Documentation Philosophy

Good documentation is as important as good code. Our goal is to create documentation that is:

-   **Accurate**: Reflects the current state of the project.
-   **Clear and Concise**: Easy to understand for users of all skill levels.
-   **Comprehensive**: Covers all aspects of the project, from installation to development.
-   **Easy to Contribute to**: The process for updating documentation should be simple and straightforward.

## Documentation Structure

The documentation is organized into several key areas:

-   **`README.md`**: The main entry point for the project. It provides a high-level overview and quick start guide.
-   **`docs/`**: This directory contains all detailed documentation.
    -   **`docs/API.md`**: The complete REST API reference.
    -   **`docs/ARCHITECTURE.md`**: A deep dive into the system's architecture.
    -   **`docs/DEPLOYMENT.md`**: A guide for deploying SupaControl to production.
    -   **`docs/DEVELOPMENT.md`**: Instructions for setting up a local development environment.
    -   **`docs/SECURITY.md`**: Security best practices and vulnerability reporting guidelines.
    -   **`docs/TROUBLESHOOTING.md`**: A guide for diagnosing and resolving common issues.
    -   **`docs/adr/`**: Architecture Decision Records (ADRs). This is where we document key architectural decisions.
-   **`CONTRIBUTING.md`**: Guidelines for contributing code and documentation.
-   **`TESTING.md`**: Information about the project's testing strategy.
-   **`CLAUDE.md` / `GEMINI.md`**: Guides for AI assistants working with the codebase.

## How to Contribute to Documentation

Contributing to the documentation is a great way to get involved in the project.

1.  **Find something to improve**: Look for typos, outdated information, or areas that are unclear. The [documentation issue template](https://github.com/qubitquilt/SupaControl/issues/new?template=documentation.md) is a good place to start.
2.  **Fork and clone the repository**: See the [CONTRIBUTING.md](../CONTRIBUTING.md) for instructions.
3.  **Create a new branch**: `git checkout -b docs/my-improvement`.
4.  **Make your changes**: Edit the relevant Markdown files.
5.  **Preview your changes**: It's a good practice to preview your changes locally before submitting a pull request. You can use a local Markdown editor like VS Code with a Markdown preview extension, or a command-line tool like `glow`.
6.  **Commit and push your changes**: Follow the [commit guidelines](../CONTRIBUTING.md#commit-guidelines) in the `CONTRIBUTING.md` file. Use the `docs:` prefix for your commit message (e.g., `docs: update installation instructions`).
7.  **Open a pull request**: Fill out the pull request template and submit your changes for review.

## Writing Style

-   **Use clear and simple language**. Avoid jargon where possible, or explain it if it's necessary.
-   **Use active voice**. For example, "The controller creates a Job" is better than "A Job is created by the controller."
-   **Use code blocks for code and commands**. Use the appropriate language identifier for syntax highlighting (e.g., ` ```yaml`).
-   **Use diagrams to illustrate complex concepts**. Mermaid diagrams are preferred as they can be embedded directly in Markdown.
-   **Keep it up-to-date**. If you are making a code change that affects the documentation, update the documentation in the same pull request.

## A Note on AI-Generated Documentation

While AI assistants can be helpful for generating documentation, it is crucial that all AI-generated content is carefully reviewed for accuracy and clarity. Do not blindly copy and paste AI-generated text. Use it as a starting point, but always verify the information against the codebase and your own understanding of the project.
