export async function resolveConflictActions(conflicts, prompt) {
  const uniqueConflicts = uniqueTargetConflicts(conflicts);
  const actions = {};
  let remainingAction = '';

  for (let index = 0; index < uniqueConflicts.length; index += 1) {
    const conflict = uniqueConflicts[index];
    let action = remainingAction;
    if (!action) {
      const decision = await prompt(conflict, index + 1, uniqueConflicts.length);
      action = normalizedAction(decision?.action);
      if (!action) return null;
      if (decision?.applyToRemaining) remainingAction = action;
    }
    actions[conflict.targetId] = action;
  }

  return actions;
}

export function uniqueTargetConflicts(conflicts = []) {
  const seen = new Set();
  return conflicts.filter((conflict) => {
    const target = String(conflict?.targetId ?? '').trim();
    if (!target || seen.has(target)) return false;
    seen.add(target);
    return true;
  });
}

function normalizedAction(value) {
  const action = String(value ?? '').trim().toLowerCase();
  return action === 'overwrite' || action === 'skip' ? action : '';
}
