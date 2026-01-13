# Revised Rule-Based Testing Rules

## Summary of Revisions

This document presents the **revised and formalized** rules for testing the combat damage calculator. The rules have been refined for clarity, mathematical precision, and testability.

---

## Complete Rule Set

### **P1 — Probability Validity**

All distributions must be valid probability distributions.

**Formal Definition:**
```
∀ k ∈ Distribution:
    p(k) ≥ -ε

Σ p(k) ∈ [1-ε, 1+ε]
```

**Test Type:** Invariant  
**Applies To:** `HitDist`, `WoundDist`, `PenDist`, `DestroyedDist`

---

### **P2 — Averages Must Match Distributions**

Reported averages must equal the expected values computed from their distributions.

**Formal Definition:**
```
AverageHits ≈ Σ (k × HitDist[k])
AverageDestroyed ≈ Σ (k × DestroyedDist[k])
```

**Test Type:** Invariant  
**Tolerance:** ε = 1e-9

---

### **P3 — Destroyed Models Are Bounded**

#### **P3a — Non-Negative Bound**

Destroyed models cannot be negative.

**Formal Definition:**
```
Destroyed ≥ -ε
∀ k ∈ DestroyedDist: k ≥ 0
```

**Test Type:** Invariant

#### **P3b — Non-Spillover Upper Bound**

Without spillover mechanics, destroyed models are bounded by attacks and targets.

**Formal Definition:**
```
If !DevastatingWounds:
    Destroyed ≤ min(totalAttacks, targetModelCount)
```

**Test Type:** Invariant 

---

### **P4 — Reroll Monotonicity**

Better reroll mechanics produce equal or better results.

**Formal Definition:**
```
For Hits:
    RerollFail ≥ RerollOnes ≥ RerollNone

For Wounds:
    RerollFail ≥ RerollOnes ≥ RerollNone
    (holding hit distribution constant)
```

**Test Type:** Comparison (ordered sequence)  
**Test Sequence:** `[RerollNone, RerollOnes, RerollFail]`

---

### **P5 — Parameter Monotonicity (Directional)**

Parameters affect output in predictable directions.

#### **Attacker Parameters** (increase output)
- `Attacker.Count`
- `Attacks` (expected value)
- `Strength`
- `AP`
- `Damage`
- `SustainedHits`
- `LethalHits`
- `DevastatingWounds`
- `Torrent`
- `HitModifier` (positive)
- `WoundModifier` (positive)

#### **Defender Parameters** (decrease output)
- `Toughness`
- `Save` (lower is better)
- `Invulnerable` (lower is better)
- `WoundsPerModel`
- `FeelNoPain` (lower is better)
- `HasCover`

**Test Type:** Comparison (pairwise)  
**Method:** Test increasing values for each parameter independently

**Example Test Values:**
- Attacker.Count: `[5, 10, 20, 50]`
- Strength: `[3, 4, 6, 8]`
- Toughness: `[3, 4, 6, 8]`
- WoundsPerModel: `[1, 2, 3, 5]`

---

### **P6 — Pipeline Isolation**

Parameters at stage N must not affect distributions at stages < N.

**Pipeline Stages:**
```
Hits → Wounds → Failed Saves → Damage → Destroyed
```

**Formal Definition:**
```
Changing parameter at stage N:
    Distributions at stages < N remain unchanged
```

**Examples:**
- `WoundModifier` must not affect `HitDist`
- `SaveReroll` must not affect `HitDist` or `WoundDist`
- `FeelNoPain` must not affect `PenDist`
- Target parameters must not affect `HitDist`

**Test Type:** Comparison (pairwise)  
**Method:** Compare upstream distributions when changing downstream parameters

---

### **P7 — Zero-Input Invariants**

Zero inputs produce deterministic zero outputs.

**Formal Definition:**
```
If Attacker.Count == 0:
    All averages == 0
    All distributions == {0: 1}

If Target.Count == 0:
    Destroyed == 0
```

**Test Type:** Invariant

---

### **P8 — Determinism**

Identical inputs always produce identical outputs.

**Formal Definition:**
```
∀ requests r1, r2:
    r1 == r2 → CalculateDamageCore(r1) == CalculateDamageCore(r2)
```

**Test Type:** Invariant  
**Method:** Run same request multiple times (n=5), compare all outputs

---

### **P9 — Distribution Support Bounds**

Distribution keys must be within physically possible ranges.

**Formal Definition:**
```
HitDist keys ∈ [0, maxPossibleHits]
WoundDist keys ∈ [0, maxPossibleWounds]
PenDist keys ∈ [0, maxPossiblePenetrations]
DestroyedDist keys ∈ [0, maxPossibleDestroyed]

where:
    maxPossibleHits = f(Attacker.Count, Attacks, SustainedHits)
    maxPossibleDestroyed = Target.Count (if !DevastatingWounds)
    maxPossibleDestroyed = unbounded (if DevastatingWounds)
```

**Test Type:** Invariant  
**Special Case:** With `DevastatingWounds`, destroyed can exceed target count

---

### P10 — Stochastic Dominance (Distributional)
**Mathematical Definition:** First-Order Stochastic Dominance (FSD).

Let $A$ be the random variable for destroyed models with "Improved" parameters.
Let $B$ be the random variable for destroyed models with "Baseline" parameters.

$A$ has first-order stochastic dominance over $B$ ($A \succeq_{1} B$) if:
$$F_A(x) \le F_B(x) \quad \forall x$$
Where $F(x) = P(X \le x)$ is the Cumulative Distribution Function (CDF).

**Strict Growth Condition:**
For a test to qualify as "Strictly Growing," two conditions must be met:
1.  **Monotonicity:** The "Improved" CDF must never exceed the "Baseline" CDF at any point ($F_A(x) \le F_B(x)$).
2.  **Differentiation:** There must be at least one outcome $x$ where the probability of achieving strictly better results is higher ($F_A(x) < F_B(x)$).

**Applicable Parameters:**
* **Attacker (Expect $\uparrow$):** `Count`, `Attacks`, `Strength`, `AP`, `Damage`, `Hit/Wound Modifiers`, `Rerolls`.
* **Defender (Expect $\downarrow$):** `Toughness`, `Save`, `Invulnerable`, `FNP`, `WoundsPerModel`.

## Test Classification

### Invariant Tests (Direct Validation)
- **P1** — Probability Validity
- **P2** — Averages Match Distributions
- **P3a** — Non-Negative Destroyed
- **P3b** — Non-Spillover Bound
- **P7** — Zero-Input Invariants
- **P8** — Determinism
- **P9** — Distribution Support Bounds

### Comparison Tests (Pairwise/Sequential)
- **P3.5** — Spillover Monotonicity
- **P4** — Reroll Monotonicity
- **P5** — Parameter Monotonicity
- **P6** — Pipeline Isolation

---

## Key Improvements from Original Rules

1. **Formalized mathematical notation** for all rules
2. **Explicit test types** (invariant vs. comparison)
3. **Concrete test values** for comparison tests
4. **Clear exception handling** (e.g., P3b with devastating wounds)
5. **Pipeline stage definitions** for P6
6. **Comprehensive parameter classification** for P5
7. **Determinism testing methodology** for P8
8. **Support bounds clarification** for P9

---

## Testing Strategy

### For Invariant Tests
1. Generate test request
2. Execute `CalculateDamageCore(req)`
3. Validate property holds on result

### For Comparison Tests
1. Generate base test request
2. Create variant by changing one parameter
3. Execute both: `result1`, `result2`
4. Compare results according to rule

### Large Number Testing
All rules should be tested with:
- **Large attacker counts** (100+)
- **Large target counts** (50+)
- **High variance dice** (multiple d6 rolls)
- **Combined edge cases** (all abilities enabled)

---

## Implementation Notes

### Epsilon Tolerance
```go
const epsilon = 1e-9
```

Used for all floating-point comparisons to account for arithmetic imprecision.

### Distribution Comparison
Two distributions are equal if:
```go
len(dist1) == len(dist2) &&
∀ k: |dist1[k] - dist2[k]| ≤ ε
```

### Monotonicity Comparison
A sequence is monotonic increasing if:
```go
∀ i: values[i+1] ≥ values[i] - ε
```

---

## Coverage Matrix

| Rule | Base Case | Large Scale | Rerolls | Special Abilities | Edge Cases |
|------|-----------|-------------|---------|-------------------|------------|
| P1   | ✓         | ✓           | ✓       | ✓                 | ✓          |
| P2   | ✓         | ✓           | ✓       | ✓                 | ✓          |
| P3a  | ✓         | ✓           | ✓       | ✓                 | ✓          |
| P3b  | ✓         | ✓           | ✗       | ✗                 | ✓          |
| P3.5 | ✗         | ✓           | ✗       | ✗                 | ✓          |
| P4   | ✓         | ✓           | ✓       | ✓                 | ✓          |
| P5   | ✓         | ✓           | ✓       | ✓                 | ✓          |
| P6   | ✓         | ✓           | ✓       | ✓                 | ✓          |
| P7   | ✗         | ✗           | ✗       | ✗                 | ✓          |
| P8   | ✓         | ✓           | ✓       | ✓                 | ✓          |
| P9   | ✓         | ✓           | ✓       | ✓                 | ✓          |

**Legend:**
- ✓ = Rule applies and is tested
- ✗ = Rule does not apply or is skipped
- Special Abilities = LethalHits, SustainedHits, DevastatingWounds
- Edge Cases = Zero inputs, exact divisions, boundary conditions

---

## Conclusion

These revised rules provide a **complete, mathematically rigorous framework** for validating the combat damage calculator. The combination of invariant and comparison tests ensures comprehensive coverage without requiring exponential test cases.

The rules are designed to:
- **Catch implementation errors** early
- **Document expected behavior** formally
- **Scale with complexity** efficiently
- **Provide confidence** in correctness
