# GitHub Commit Rules

## Commit Message Format

All commit messages should follow the conventional commit format:

```
type(scope): description

body (optional)

footer (optional)
```

## Types

- **feat**: New feature
- **fix**: Bug fix
- **docs**: Documentation changes
- **style**: Code style changes (formatting, etc.)
- **refactor**: Code refactoring
- **test**: Adding or updating tests
- **chore**: Maintenance tasks

## Scopes

Common scopes include:
- **agent**: Agent-related changes
- **tool**: Tool-related changes
- **channel**: Channel-related changes
- **provider**: LLM provider-related changes
- **config**: Configuration-related changes
- **webui**: Web UI-related changes
- **cli**: Command-line interface changes
- **utils**: Utility functions changes

## Examples

### Feature
```
feat(tool): add interaction tool

- Rename MouseTool to InteractionTool
- Add support for keyboard input and mouse clicks
- Update tool registration in agent loop
```

### Bug Fix
```
fix(agent): resolve message routing issue

- Fix agent dispatcher to properly route messages
- Ensure only mentioned agents receive messages
```

### Documentation
```
docs(readme): update agent configuration guide

- Add examples for multi-agent setup
- Document @mention functionality
```

## Best Practices

1. Keep commit messages concise and descriptive
2. Use present tense ("add" not "added")
3. Limit the subject line to 50 characters
4. Use lowercase for the subject line
5. Separate subject from body with a blank line
6. Wrap body lines at 72 characters
7. Use the body to explain what and why, not how
8. Reference issues in the footer if applicable

## Branch Naming

- Feature branches: `feat/feature-name`
- Bug fix branches: `fix/bug-description`
- Hotfix branches: `hotfix/issue-description`
- Release branches: `release/version-number`

## Pull Request Guidelines

- One feature per pull request
- Include tests for new functionality
- Update documentation if necessary
- Reference related issues
- Keep PR descriptions clear and concise