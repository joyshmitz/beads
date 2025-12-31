# Incident Report: Mermaid Diagram Not Rendering on GitHub Pages

**Date**: 2025-12-31
**Status**: UNRESOLVED
**Severity**: Medium (documentation quality)
**Site**: https://joyshmitz.github.io/beads/architecture

---

## Executive Summary

Mermaid diagram on the Architecture page renders correctly in local builds but displays as a raw code block on the deployed GitHub Pages site. Multiple deployment attempts and cache clearing did not resolve the issue.

---

## The Problem

### Expected Behavior
```
┌─────────────────────────────────────┐
│  Mermaid SVG Diagram                │
│  (rendered flowchart)               │
└─────────────────────────────────────┘
```

### Actual Behavior (Deployed)
```
┌─────────────────────────────────────┐
│ flowchart TD                        │
│     subgraph GIT["Layer 1:..."]     │
│ (raw code with syntax highlighting) │
└─────────────────────────────────────┘
```

---

## Root Cause Analysis

### Key Evidence

| Aspect | Local Build | GitHub Actions Build |
|--------|-------------|---------------------|
| HTML output | No `language-mermaid` class | Has `language-mermaid` class |
| JS bundle | Contains `(0,r.jsx)(n.mermaid,{value:'...'})` | Contains raw code in `<pre>` |
| Behavior | Mermaid component (client-side render) | Code block (syntax highlighted) |

### The Core Issue

**The MDX remark plugin for mermaid is NOT transforming code blocks in GitHub Actions builds.**

When Docusaurus processes markdown:
- **Correct**: ` ```mermaid ` → `<Mermaid value="...">` component → JS bundle → client-side SVG
- **Broken (CI)**: ` ```mermaid ` → `<pre class="language-mermaid">` → static HTML code block

---

## Configuration (Verified Correct)

### docusaurus.config.ts
```typescript
// Enable Mermaid diagrams in markdown
markdown: {
  mermaid: true,
},
themes: ['@docusaurus/theme-mermaid'],
```

### package.json
```json
"@docusaurus/theme-mermaid": "^3.9.2"
```

### package-lock.json
```json
"node_modules/@docusaurus/theme-mermaid": {
  "version": "3.9.2",
  "resolved": "https://registry.npmjs.org/@docusaurus/theme-mermaid/-/theme-mermaid-3.9.2.tgz"
}
```

---

## Attempted Fixes (All Failed)

### 1. React Downgrade (PR #45)
- Changed React 19 → 18.3.1
- Result: No effect on mermaid

### 2. Disable future.v4 (PR #46)
- Commented out `future: { v4: true }`
- Result: No effect on mermaid

### 3. Clear Docusaurus Cache
```yaml
- name: Clear build caches
  working-directory: website
  run: rm -rf .docusaurus build
```
- Result: No effect

### 4. Clean npm Cache
```yaml
- name: Clean npm cache
  run: npm cache clean --force
```
- Result: No effect

### 5. Multiple Redeploys
Triggered 4+ deploys with different main.js hashes:
- `main.ad400dcb.js`
- `main.4d4ef15d.js`
- `main.8aa0f470.js`
- `main.1efe2a43.js`

All show mermaid as code block.

---

## Hypotheses (Not Yet Tested)

### H1: npm ci Installs Different Dependencies
GitHub Actions `npm ci` might install different versions despite package-lock.json.
- **Test**: Add `npm ls @docusaurus/theme-mermaid` to workflow logs

### H2: TypeScript Config Not Parsed
The `.ts` config file might not be compiled correctly in CI.
- **Test**: Convert to `docusaurus.config.js` temporarily

### H3: Plugin Load Order Issue
The mermaid remark plugin might load AFTER the code block processor.
- **Test**: Check Docusaurus plugin registration order in build logs

### H4: Environment Variable Difference
Some env var affects plugin behavior.
- **Test**: Compare `process.env` in local vs CI builds

### H5: Node Version Mismatch
Local Node version differs from CI.
- **Test**: Run `node --version` locally and compare to workflow

### H6: theme-mermaid Not Actually Used
The theme might be installed but not registered as a remark plugin.
- **Test**: Add debug logging to see which plugins are registered

---

## Technical Deep Dive

### How Mermaid Should Work in Docusaurus

1. `@docusaurus/theme-mermaid` registers a remark plugin
2. Remark plugin finds ` ```mermaid ` code blocks in MDX
3. Plugin transforms them to `<Mermaid value="...">` JSX
4. Mermaid component is bundled into JS
5. Client-side JS renders SVG when page loads

### Where It's Breaking

The remark plugin transformation (step 2→3) is NOT happening in CI.
The code block goes directly to Prism syntax highlighter instead.

### Evidence from Build Output

**Local build JS (dc7abf1f.aab93fd1.js)**:
```javascript
(0,r.jsx)(n.mermaid,{value:'flowchart TD\n    subgraph GIT["🗂️ Layer 1...
```

**Deployed HTML**:
```html
<div class="language-mermaid codeBlockContainer_Ckt0">
  <pre class="prism-code language-mermaid">
    <code>flowchart TD
    subgraph GIT["🗂️ Layer 1: Git Repository"]
```

---

## Files Modified During Investigation

1. **`.github/workflows/deploy-docs.yml`**
   - Added: `npm cache clean --force`
   - Added: `rm -rf .docusaurus build`

2. **`website/docusaurus.config.ts`**
   - Added trigger comments (cosmetic)

3. **`website/.buildtrigger`**
   - Created to force workflow trigger

---

## Commits Created

| Commit | Message | Effect |
|--------|---------|--------|
| `daaecab0` | chore: force redeploy for mermaid fix | No trigger (empty commit) |
| `b4585b26` | chore: trigger redeploy for mermaid fix | Deployed, still broken |
| `09c97a66` | fix(ci): trigger rebuild with cache clear | Deployed, still broken |
| `d0f880d4` | fix(ci): clear docusaurus cache before build | No trigger (workflow file only) |
| `eb17a3ef` | fix(ci): force clean npm cache before install | No trigger (workflow file only) |
| `4189ca0d` | chore: trigger build | Deployed, still broken |

---

## Next Steps for Investigation

### Immediate Actions
1. **Check GitHub Actions build logs** for any warnings about mermaid or theme loading
2. **Add verbose logging** to workflow to capture installed packages
3. **Compare node/npm versions** between local and CI

### Potential Fixes to Try
1. **Use workflow_dispatch** to manually trigger with debug
2. **Try docusaurus.config.js** instead of .ts
3. **Explicitly import theme-mermaid** in config
4. **Check if remark-mermaid plugin is registered**

### Debug Commands to Add to Workflow
```yaml
- name: Debug dependencies
  working-directory: website
  run: |
    npm ls @docusaurus/theme-mermaid
    npm ls mermaid
    cat node_modules/@docusaurus/theme-mermaid/package.json | head -20
    node --version
    npm --version
```

---

## Environment Details

| Component | Local | GitHub Actions |
|-----------|-------|----------------|
| Node | ? | 20 |
| npm | ? | ? |
| Docusaurus | 3.9.2 | 3.9.2 |
| theme-mermaid | 3.9.2 | 3.9.2 (expected) |
| React | 18.3.1 | 18.3.1 |
| OS | Linux | ubuntu-latest |

---

## Related Files

- Source: `website/docs/architecture/index.md`
- Config: `website/docusaurus.config.ts`
- Workflow: `.github/workflows/deploy-docs.yml`
- Package: `website/package.json`
- Lock: `website/package-lock.json`

---

## Conclusion

The issue is that GitHub Actions builds do not apply the mermaid remark plugin transformation, causing mermaid code blocks to be rendered as syntax-highlighted code instead of Mermaid components. Local builds work correctly. The exact cause remains unknown despite clearing caches and forcing fresh installs.

**Key insight**: This is a build-time transformation issue, not a runtime or caching issue.
