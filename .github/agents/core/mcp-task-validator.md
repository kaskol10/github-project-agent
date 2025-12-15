# Task Validation Schema for MCP

## Purpose
This schema defines the structure and validation rules for tasks processed by an MCP (Model Context Protocol). The MCP should validate task completeness, consistency, and quality before accepting submissions.

---

## Task Metadata

```
Priority: [Critical | High | Medium | Low]
Complexity: [Complex | Moderate | Trivial]
Days (optional): [Number of estimated days]
Start Date: [YYYY-MM-DD]
End Date: [YYYY-MM-DD]
```

### Validation Rules for Metadata:
- **Priority**: Must be one of the four defined levels (required)
- **Complexity**: Must be one of the three defined levels (required)
- **Days**: Must be a positive integer if provided (optional)
- **Start Date**: Must be a valid date in YYYY-MM-DD format and not in the past (required)
- **End Date**: Must be a valid date in YYYY-MM-DD format and equal to or after Start Date (required)

---

## Task Content Structure

### Description
[Explain what needs to be done in this task, relevant context, the value/impact that adds and any dependencies or blockers that could have with other tasks]

**Validation Rules:**
- Should clearly explain the "what" and "why"
- Should mention dependencies or blockers if applicable
- Should include business value or impact statement

### Scope
[Define what is included in this task and what is explicitly out of scope. Clarify any boundaries, assumptions, or constraints that affect the work]

**Validation Rules:**
- Must define at least one item that IS in scope
- Must explicitly state at least one item that is OUT of scope
- Should mention key constraints or assumptions

### Acceptance Criteria
[Specific, measurable outcome or deliverable]
- [ ] Requirement 1
- [ ] Requirement 2
- [ ] Requirement 3

**Validation Rules:**
- Each criterion should be specific and measurable (avoid vague terms like "good", "nice", "better")
- Criteria should be testable/verifiable
- Criteria should not be interdependent on one another
- Use checkbox format `- [ ]` for each requirement
- Avoid using "and" within a single criterion (split into multiple)
- Examples: "API endpoint returns response in under 200ms", "All edge cases documented", "Unit test coverage above 80%"

### Definition of Done (optional)
[Checklist of completion requirements beyond acceptance criteria, such as documentation, code review, testing, deployment, etc.]
- [ ] Item 1
- [ ] Item 2

**Validation Rules:**
- Optional section
- Use checkbox format for each done item
- Should include non-functional requirements like documentation, testing, review processes

### Additional Notes
[Implementation considerations, technical details, design references, or any other resources needed to complete the task]

**Validation Rules:**
- Recommended but optional
- If provided, should contain actionable information
- Can include links, references, or technical specifications

---

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
- [ ] Requirement 1
- [ ] Requirement 2
- [ ] Requirement 3

### Definition of Done (optional)
- [ ] Item 1
- [ ] Item 2

### Additional Notes
[Optional implementation details, resources, or references]
```

---

## MCP Validation Checklist

When validating a task, the MCP should check:

**Metadata:**
- ✓ Priority is one of four valid values
- ✓ Complexity is one of three valid values
- ✓ Start Date and End Date are valid formats (YYYY-MM-DD)
- ✓ End Date ≥ Start Date
- ✓ Days (if provided) is a positive integer

**Content:**
- ✓ Description exists and clearly explains the "what" and "why" with business value
- ✓ Scope defines what's IN and explicitly OUT of scope
- ✓ Acceptance Criteria items are specific, measurable, and testable
- ✓ No vague language in criteria (good, nice, better, etc.)
- ✓ Title is descriptive and concise
- ✓ Definition of Done (if provided) includes non-functional requirements

**Suggestions for Improvement:**
- If Acceptance Criteria are vague, suggest making them more measurable
- If Description lacks impact statement, suggest adding business value
- If Scope is incomplete, suggest clarifying boundaries
- If Definition of Done is missing, suggest including operational completion items