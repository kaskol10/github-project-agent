# Task Format Guidelines

This document defines the format requirements for all GitHub issues/tasks in this project, following the MCP Task Validation Schema.

> **Note**: For the complete MCP Task Validation Schema, see `.github/agents/core/mcp-task-validator.md`

## Format Rules

### Required Metadata

All tasks must include the following metadata at the top:

- **Priority**: Must be one of [Critical | High | Medium | Low]
- **Complexity**: Must be one of [Complex | Moderate | Trivial]
- **Days** (optional): Positive integer for estimated days
- **Start Date**: Valid date in YYYY-MM-DD format, not in the past
- **End Date**: Valid date in YYYY-MM-DD format, must be ≥ Start Date

### Required Sections

All tasks must include the following sections:

1. **Description**: 
   - Must explain "what" and "why"
   - Must include business value/impact statement
   - Must mention dependencies or blockers if applicable

2. **Scope**: 
   - Must define at least one item that IS in scope
   - Must explicitly state at least one item that is OUT of scope
   - Should mention key constraints or assumptions

3. **Acceptance Criteria**: 
   - Each criterion must be specific and measurable
   - Must be testable/verifiable
   - Criteria should not be interdependent
   - Use checkbox format: `- [ ] Requirement`
   - Avoid vague terms like "good", "nice", "better"
   - Avoid "and" within a single criterion (split into multiple)

4. **Definition of Done** (optional but recommended):
   - Use checkbox format: `- [ ] Item`
   - Should include non-functional requirements (documentation, testing, review, deployment)

5. **Additional Notes** (optional):
   - Implementation considerations
   - Technical details
   - Design references
   - Resources needed

### Minimum Requirements

- **Description Length**: Minimum 100 characters
- **Acceptance Criteria**: At least 3 specific, measurable criteria
- **Dates**: Start Date must not be in the past, End Date ≥ Start Date

## Instructions

When creating a task, please ensure:

1. The title is clear and descriptive
2. All required metadata is present (Priority, Complexity, Start Date, End Date)
3. The description provides enough context including "what", "why", and business value
4. Scope clearly defines what's IN and OUT of scope
5. Acceptance criteria are specific, measurable, and testable
6. Definition of Done includes operational completion items
7. Dependencies or related issues are mentioned if applicable

## Complete Task Template

```markdown
# [Task Title]

**Priority:** [Critical | High | Medium | Low]
**Complexity:** [Complex | Moderate | Trivial]
**Days (optional):** [Number]
**Start Date:** YYYY-MM-DD
**End Date:** YYYY-MM-DD

---

### Description
[Explain what needs to be done, relevant context, value/impact, and any dependencies or blockers]

### Scope
[Define what is included and explicitly out of scope. Clarify boundaries, assumptions, or constraints]

### Acceptance Criteria
- [ ] Requirement 1 (specific, measurable, testable)
- [ ] Requirement 2 (specific, measurable, testable)
- [ ] Requirement 3 (specific, measurable, testable)

### Definition of Done (optional)
- [ ] Documentation updated
- [ ] Code reviewed and approved
- [ ] Tests written and passing
- [ ] Deployed to staging environment

### Additional Notes
[Optional implementation details, resources, or references]
```

## Examples

### Good Example

```markdown
# Implement OAuth2 Authentication

**Priority:** High
**Complexity:** Moderate
**Days:** 5
**Start Date:** 2025-01-15
**End Date:** 2025-01-20

---

### Description

Implement user authentication using OAuth2 with GitHub as the provider. This will allow users to sign in using their GitHub accounts and enable secure access to the application. This feature is critical for user onboarding and reduces friction in the sign-up process. 

**Dependencies**: Requires GitHub OAuth app configuration (issue #123)
**Blockers**: None

### Scope

**In Scope:**
- OAuth2 flow implementation with GitHub
- User session management
- Secure token storage
- Sign-in and sign-out functionality

**Out of Scope:**
- Multi-factor authentication
- Social login with other providers (Google, Twitter)
- Password-based authentication

### Acceptance Criteria
- [ ] User can click "Sign in with GitHub" button and initiate OAuth flow
- [ ] OAuth2 flow completes successfully and redirects user back to application
- [ ] User session is created and stored securely (JWT token with 24h expiration)
- [ ] User can sign out, which invalidates the session token
- [ ] Error handling displays user-friendly messages for failed authentication attempts
- [ ] API endpoint `/auth/github/callback` returns 200 status code on success

### Definition of Done
- [ ] Code reviewed and approved by at least 2 team members
- [ ] Unit tests written with >80% coverage
- [ ] Integration tests for OAuth flow
- [ ] Documentation updated with setup instructions
- [ ] Deployed to staging environment and tested
- [ ] Security review completed

### Additional Notes
- Use `github.com/google/go-github` library for GitHub API integration
- Store tokens in secure HTTP-only cookies
- Reference: [GitHub OAuth Documentation](https://docs.github.com/en/apps/oauth-apps)
```

### Bad Example

```markdown
Add login
```

**Issues with this example:**
- Missing all required metadata (Priority, Complexity, Dates)
- No description explaining "what" and "why"
- No scope definition
- No acceptance criteria
- Too vague and lacks necessary information
- No business value statement

