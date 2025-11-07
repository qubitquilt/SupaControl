# supactl README Alignment Recommendations

This document provides recommendations for aligning the supactl README with the SupaControl README standards.

## Current Status

✅ **Already Correct:**
- Badge style: All badges use `for-the-badge` style
- Structure: Well-organized with clear sections
- Documentation: Comprehensive and detailed

## Recommended Changes

### 1. Add Missing Technology Badges

#### Add Docker Badge

Docker is a core dependency for local mode and should be prominently featured:

```markdown
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)
```

**Placement:** After the Go badges, before platform badges

#### Add Cobra CLI Framework Badge (Optional)

Since Cobra is mentioned in acknowledgments:

```markdown
![Cobra CLI](https://img.shields.io/badge/Cobra%20CLI-000000?style=for-the-badge&logo=go&logoColor=white)
```

### 2. Reorganize Badge Layout

Update the badge section to match SupaControl's structure:

```markdown
# Supabase Management Tools

A comprehensive toolkit for managing self-hosted Supabase instances.

**supactl** is a modern, unified CLI for managing Supabase instances both remotely (via a SupaControl server) and locally (direct Docker management).

<!-- Status Badges -->
[![Test](https://img.shields.io/github/actions/workflow/status/qubitquilt/supactl/test.yml?style=for-the-badge&label=tests)](https://github.com/qubitquilt/supactl/actions)
[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen?style=for-the-badge&logo=go&logoColor=white)](https://goreportcard.com/report/github.com/qubitquilt/supactl)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg?style=for-the-badge)](https://www.gnu.org/licenses/gpl-3.0)
[![GitHub release](https://img.shields.io/github/v/release/qubitquilt/supactl?style=for-the-badge)](https://github.com/qubitquilt/supactl/releases)

## Built With

**Technologies:**

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Go Version](https://img.shields.io/github/go-mod/go-version/qubitquilt/supactl?style=for-the-badge)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)

**Platforms:**

![Linux](https://img.shields.io/badge/Linux-FCC624?style=for-the-badge&logo=linux&logoColor=black)
![macOS](https://img.shields.io/badge/mac%20os-000000?style=for-the-badge&logo=macos&logoColor=F0F0F0)
![Windows](https://img.shields.io/badge/Windows-0078D4?style=for-the-badge&logo=windows&logoColor=white)
```

### 3. Fix Go Report Card Badge

**Current (incorrect):**
```markdown
[![Go Report Card](https://goreportcard.com/badge/github.com/qubitquilt/supactl?style=for-the-badge)](https://goreportcard.com/report/github.com/qubitquilt/supactl)
```

**Option A - Dynamic Score:**
```markdown
[![Go Report Card](https://goreportcard.com/badge/github.com/qubitquilt/supactl?style=for-the-badge)](https://goreportcard.com/report/github.com/qubitquilt/supactl)
```
*Note: Go Report Card's native badge might not support style parameter*

**Option B - Shields.io Format (Recommended):**
```markdown
[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen?style=for-the-badge&logo=go&logoColor=white)](https://goreportcard.com/report/github.com/qubitquilt/supactl)
```

### 4. Enhance Cross-References

The supactl README already references SupaControl throughout, which is excellent. Consider adding a prominent callout at the top:

```markdown
> **Part of the SupaControl Ecosystem**
>
> This CLI tool is designed to work with [SupaControl](https://github.com/qubitquilt/SupaControl), a self-hosted management platform for orchestrating multi-tenant Supabase instances on Kubernetes.
```

### 5. Add "Related Projects" Section

Add before or after the "Support & Community" section:

```markdown
## 🔗 Related Projects

- **[SupaControl](https://github.com/qubitquilt/SupaControl)** - Self-hosted management platform for orchestrating multi-tenant Supabase instances on Kubernetes
  - REST API server for centralized management
  - Web dashboard for visual management
  - Kubernetes Operator pattern implementation
  - Multi-tenant support with namespace isolation

- **supactl** (this project) - CLI tool for managing instances
  - Remote management via SupaControl API
  - Local Docker-based management
  - Cross-platform support
```

## Alignment Checklist

- [ ] Add Docker badge to technology section
- [ ] Reorganize badges into "Built With" → "Technologies" and "Platforms"
- [ ] Fix Go Report Card badge format
- [ ] Add prominent SupaControl ecosystem callout
- [ ] Add Related Projects section
- [ ] Ensure all shields.io badges have `style=for-the-badge`
- [ ] Consider adding Cobra CLI badge

## Badge Color Codes Reference

For consistency with SupaControl badges:

- **Go:** `#00ADD8` (cyan)
- **Docker:** `#0db7ed` (light blue)
- **GitHub Actions:** `#2671E5` (blue)
- **Postgres:** `#316192` (dark blue)
- **Kubernetes:** `#326ce5` (blue)
- **Helm:** `#0F1689` (dark blue)
- **React:** `#20232a` / `#61DAFB` (dark gray / cyan)
- **Vite:** `#646CFF` (purple)
- **Node.js:** `#6DA55F` (green)
- **npm:** `#CB3837` (red)

## Additional Recommendations

### Consider Adding

1. **Installation Verification Badge**
   ```markdown
   ![Install](https://img.shields.io/badge/install-script%20available-success?style=for-the-badge)
   ```

2. **Documentation Badge**
   ```markdown
   [![Documentation](https://img.shields.io/badge/docs-available-blue?style=for-the-badge)](SUPACTL_README.md)
   ```

3. **Docker Pulls (if published to Docker Hub)**
   ```markdown
   ![Docker Pulls](https://img.shields.io/docker/pulls/qubitquilt/supactl?style=for-the-badge&logo=docker&logoColor=white)
   ```

## Badge Order Recommendation

1. **Status Badges** (CI, Coverage, Quality, Release)
2. **Built With Section**
   - Technologies subsection
   - Platforms subsection
3. **Project Description**
4. **Content...**

This creates visual hierarchy and makes it easy for visitors to quickly understand:
- Project health (status badges)
- Tech stack (built with badges)
- What the project does (description)

## Notes

- All badges should use `style=for-the-badge` for visual consistency
- Maintain alphabetical or logical grouping within badge categories
- Use shields.io format for badges that need customization
- Ensure all badge links are working and point to correct resources
- Keep badge colors consistent with official brand colors where applicable

## Implementation Priority

**High Priority:**
1. Add Docker badge (core dependency)
2. Fix Go Report Card badge format
3. Reorganize badge layout

**Medium Priority:**
4. Add prominent ecosystem callout
5. Add Related Projects section

**Low Priority:**
6. Add Cobra CLI badge (optional)
7. Add additional badges (documentation, etc.)
