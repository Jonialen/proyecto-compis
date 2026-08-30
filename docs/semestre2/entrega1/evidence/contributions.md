# Contribution evidence

This evidence records only authorship observable in the repository history. It does not infer an additional contributor to satisfy the course requirement.

## Reproduce the count

Run from the repository root:

```bash
git shortlog -sne --all
```

Observed on 2026-08-30 at baseline revision `97c394b` (later documentation commits are intentionally outside this snapshot):

```text
53  jonialen <jonathanalej09@gmail.com>
17  Jonialen <jonathanalej09@gmail.com>
 9  DijanU <pad23663@uvg.edu.gt>
 1  Jonathan Díaz <77712004+Jonialen@users.noreply.github.com>
```

The command reports 80 commits across four author signatures. The first two signatures share the same email and differ only by capitalization, so they are aliases for one observable contributor identity (70 commits combined). The GitHub noreply signature contains the same `Jonialen` account handle; it is documented as another Jonathan/Jonialen alias (1 commit), rather than as a third person. `DijanU <pad23663@uvg.edu.gt>` is the second observable contributor identity (9 commits).

## Requirement limitation

The Git history therefore supports two contributor identities, not three. This evidence cannot demonstrate compliance with a three-member contribution requirement. Adding a name or attribution without a corresponding commit would be inaccurate; any future update must be regenerated from the real Git history.
