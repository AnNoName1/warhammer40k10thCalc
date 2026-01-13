# Rule-Based Testing Documentation for Combat Calculator

## Overview

This document describes the comprehensive rule-based testing strategy for the Warhammer 40k combat damage calculator. The approach uses **property-based testing** principles to verify mathematical invariants and behavioral properties rather than testing specific input-output pairs.

## Why Rule-Based Testing?

The calculator accepts approximately **30 input parameters** and produces **multiple probability distributions** (hits, wounds, penetrations, destroyed models). Traditional unit testing would require an exponential number of test cases to achieve adequate coverage. Rule-based testing instead validates **mathematical properties** that must hold for **all valid inputs**.

## Testing Philosophy

### TDD Approach
The implementation is deliberately hidden, and tests are written against the public interface:
- `CombatSimulationRequest` (input)
- `SimulationResult` (output)

This ensures tests validate **behavior** rather than **implementation details**.

### Test Categories

1. **Invariant Tests (P1, P2, P7, P8, P9)**: Properties that must always hold
2. **Comparison Tests (P3.5, P4, P5, P6)**: Properties verified by comparing two related cases

## Rule Definitions

### P1 — Probability Validity

**Mathematical Definition:**
```
∀ k ∈ Distribution:
    p(k) ≥ -ε
Σ p(k) ∈ [1-ε, 1+ε]
```

**What it tests:**
- All probability values are non-negative
- All distributions sum to 1.0 (within floating-point tolerance)

**Implementation:**
- Tests all four distributions: `HitDist`, `WoundDist`, `PenDist`, `DestroyedDist`
- Uses epsilon tolerance (`1e-9`) for floating-point comparisons
- Validates across multiple scenarios: base case, large scale, with rerolls, with special abilities

**Why it matters:** Invalid probability distributions indicate fundamental mathematical errors in the calculation engine.

---

### P2 — Averages Must Match Distributions

**Mathematical Definition:**
```
AverageHits ≈ Σ (k × HitDist[k])
AverageDestroyed ≈ Σ (k × DestroyedDist[k])
```

**What it tests:**
- The reported average values match the expected values computed from their distributions

**Implementation:**
- Calculates expected value from distribution: `E[X] = Σ k·p(k)`
- Compares with reported averages within epsilon tolerance
- Tests with various damage and attack patterns (dice notation)

**Why it matters:** Ensures consistency between summary statistics and full distributions.

---

### P3 — Destroyed Models Are Bounded

#### P3a — Non-Negative Destroyed

**Mathematical Definition:**
```
Destroyed ≥ -ε
```

**What it tests:**
- Average destroyed is non-negative
- All distribution keys are non-negative

**Why it matters:** Negative destroyed models are physically impossible.

#### P3b — Non-Spillover Bound

**Mathematical Definition:**
```
If !DevastatingWounds:
    Destroyed ≤ min(totalAttacks, targetModelCount)
```

**What it tests:**
- Without devastating wounds, destroyed models cannot exceed the smaller of:
  - Total possible attacks
  - Total target models

**Implementation:**
- Checks maximum key in `DestroyedDist`
- Tests with varying target counts

**Why it matters:** Prevents unrealistic overkill without spillover mechanics.

### P4 — Monotonicity Properties (Rerolls)

**Mathematical Definition:**
```
For Hits:
    RerollFail ≥ RerollOnes ≥ RerollNone

For Wounds:
    Same, holding hit distribution constant
```

**What it tests:**
- Better reroll mechanics always produce equal or better results
- Applies to both hit and wound rolls

**Implementation:**
- **Comparison test**: Tests reroll progression `[None, Ones, Fail]`
- Verifies monotonic increase in average hits/wounds
- Tests independently for hit and wound phases

**Why it matters:** Ensures reroll mechanics provide consistent benefits.

---

### P5 — Parameter Monotonicity (Directional)

**Mathematical Definition:**

**Attacker parameters** (should increase output):
- `Attacker.Count`, `Attacks`, `Strength`, `AP`, `Damage`
- `SustainedHits`, `LethalHits`, `DevastatingWounds`, `Torrent`
- `HitModifier`, `WoundModifier`

**Defender parameters** (should decrease output):
- `Toughness`, `Save`, `Invulnerable`, `WoundsPerModel`
- `FeelNoPain`, `HasCover`

**What it tests:**
- Improving attacker stats increases destroyed models
- Improving defender stats decreases destroyed models

**Implementation:**
- **Comparison tests**: For each parameter, test increasing values
- Attacker tests: `[5, 10, 20, 50]` for count, `[3, 4, 6, 8]` for strength, etc.
- Defender tests: `[3, 4, 6, 8]` for toughness, `[1, 2, 3, 5]` for wounds, etc.
- Verifies monotonic behavior in expected direction

**Why it matters:** Validates that all parameters affect outcomes in the correct direction.

---

### P6 — Pipeline Isolation

**Mathematical Definition:**
```
Pipeline: Hits → Wounds → Failed Saves → Damage → Destroyed

Changing parameter at stage N must not affect stages < N
```

**Examples:**
- `WoundModifier` must not affect `HitDist`
- `SaveReroll` must not affect `HitDist` or `WoundDist`
- `FeelNoPain` must not affect `PenDist`

**What it tests:**
- Each stage of the combat pipeline is independent
- Later stages don't retroactively affect earlier stages

**Implementation:**
- **Comparison tests**: Runs two cases differing only in a downstream parameter
- Compares upstream distributions for equality
- Tests multiple pipeline boundaries

**Why it matters:** Ensures proper separation of concerns and prevents cascading calculation errors.

---

### P7 — Zero-Input Invariants

**Mathematical Definition:**
```
If Attacker.Count == 0:
    All averages == 0
    All distributions == {0: 1}

If Target.Count == 0:
    Destroyed == 0
```

**What it tests:**
- Edge case handling for zero inputs
- Distributions degenerate correctly to certain outcomes

**Implementation:**
- Tests with `Attacker.Count = 0`
- Tests with `Target.Count = 0`
- Verifies all outputs are zero or {0: 1} distributions

**Why it matters:** Validates edge case handling and prevents division-by-zero errors.

---

### P8 — Determinism

**Mathematical Definition:**
```
Same input → Same output (always)
```

**What it tests:**
- Calculator produces identical results for identical inputs
- No hidden randomness or state

**Implementation:**
- Runs same request 5 times
- Compares all outputs for exact equality (within epsilon)
- Tests across multiple scenarios

**Why it matters:** Ensures reproducibility and absence of unintended randomness.

---

### P9 — Distribution Support Bounds

**Mathematical Definition:**
```
HitDist keys ∈ [0, maxPossibleHits]
DestroyedDist keys ∈ [0, maxPossibleDestroyed]
```

**What it tests:**
- Distribution keys are within physically possible ranges
- No impossible outcomes have non-zero probability

**Implementation:**
- Calculates theoretical maximums based on input parameters
- Validates all distribution keys fall within bounds
- Special handling for devastating wounds (allows exceeding target count)

**Why it matters:** Prevents distributions from including impossible outcomes.

---

### P10 — Stochastic Dominance (Distributional)

**What it tests:**
Verifies that improving input parameters shifts the **entire probability distribution** favorably, rather than just increasing the average value. It ensures that a "stronger" attacker never has a higher probability of failure than a "weaker" one.

**Implementation:**
* **CDF Construction:** Converts the Probability Mass Function (PMF) into a Cumulative Distribution Function (CDF).
* **Comparison:** Iterates through the domain of all possible damage values.
* **Assertion:** Checks that $CDF_{better}(x) \le CDF_{worse}(x)$ for all $x$.

**Why it matters:**
Prevents "high variance" bugs where increasing a stat (like Damage) might inadvertently increase the probability of scoring zero kills (e.g., due to overflow or improper spill-over logic). It guarantees the integrity of the risk profile.

---

## Stress Testing with Large Numbers

In addition to rule validation, the test suite includes **stress tests** with large parameter values:

- **Very large attacker count** (1000 models)
- **Very large target count** (500 models)
- **High variance attacks** (6d6 attacks per model)
- **All abilities enabled** (combined edge cases)

These tests ensure the calculator scales correctly and maintains mathematical properties even with extreme inputs.

## Test Execution

### Running All Tests
```bash
go test -v
```

### Running Specific Rule Tests
```bash
go test -v -run TestP1_ProbabilityValidity
go test -v -run TestP5_ParameterMonotonicity
```

### Running Without Stress Tests
```bash
go test -v -short
```

### Running Stress Tests Only
```bash
go test -v -run TestStressTest
```

## Test Configuration

### Epsilon Tolerance
```go
const epsilon = 1e-9
```

This tolerance accounts for floating-point arithmetic imprecision. Adjust if needed based on calculation precision requirements.

### Test Data Generators

**`generateBaseRequest()`**: Creates a standard balanced scenario
- 10 attackers with 2 attacks each
- BS 3+, S4, AP-1, D1
- 10 targets with T4, 3+ save, 2 wounds

**`generateLargeScaleRequest()`**: Creates a large-scale scenario
- 100 attackers with 6 attacks each
- 50 targets

These generators ensure consistent test data across all rule validations.

## Extending the Test Suite

### Adding New Rules

1. **Define the mathematical property** in this document
2. **Implement the test function** following the naming convention `TestPX_RuleName`
3. **Use comparison testing** for monotonicity/isolation rules (P3.5, P4, P5, P6)
4. **Use invariant testing** for universal properties (P1, P2, P7, P8, P9)

### Adding New Test Cases

Add new scenarios to existing test functions by extending the `testCases` slice:

```go
testCases := []struct {
    name string
    req  CombatSimulationRequest
}{
    {"Base case", generateBaseRequest()},
    {"Your new case", func() CombatSimulationRequest {
        req := generateBaseRequest()
        // Modify req as needed
        return req
    }()},
}
```

## Interpretation of Test Failures

### P1 Failures
- **Negative probabilities**: Likely underflow or incorrect probability calculation
- **Sum ≠ 1**: Missing cases in distribution or normalization error

### P2 Failures
- **Average mismatch**: Inconsistency between distribution calculation and average calculation
- Check if both are computed from the same source

### P3/P3.5 Failures
- **Bounds exceeded**: Check spillover logic and damage application
- **Monotonicity violation**: Review how increasing attacks affects damage calculation

### P4/P5 Failures
- **Monotonicity violation**: Parameter not being applied correctly or in wrong direction
- Check the combat resolution pipeline for the affected parameter

### P6 Failures
- **Pipeline isolation broken**: Later stage is affecting earlier stage
- Review calculation order and data flow

### P7 Failures
- **Zero handling**: Check for division by zero or missing edge case handling

### P8 Failures
- **Non-determinism**: Hidden randomness or uninitialized state
- Check for time-based seeds or uninitialized variables

### P9 Failures
- **Out-of-bounds keys**: Distribution includes impossible outcomes
- Review maximum value calculations

### P10 Failures
**Dominance Violation ($F_{new} > F_{old}$)**
* **Meaning:** The "improved" stat resulted in a *higher* probability of achieving a low result ($ \le x$).
* **Likely Causes:**
    * **Spill-over Logic:** Higher damage per shot causing fewer distinct kills due to lack of spill-over.
    * **Reroll Regressions:** Reroll logic accidentally accepting a lower result than the original roll.
    * **Overflow:** Integer wrapping on high damage counts.

**No Strict Improvement ($F_{new} \equiv F_{old}$)**
* **Meaning:** The parameter change resulted in an identical distribution.
* **Likely Causes:**
    * **Thresholds:** Increasing Strength from 6 to 7 against Toughness 4 (both wound on 3+).
    * **Caps:** Improving AP beyond the target's Invulnerable Save cap.
    * **Implementation Error:** The parameter is ignored in the calculation pipeline.


## Best Practices

1. **Always use epsilon tolerance** for floating-point comparisons
2. **Test with large numbers** to expose scaling issues
3. **Use comparison tests** for relative properties (monotonicity, isolation)
4. **Use invariant tests** for absolute properties (validity, determinism)
5. **Document assumptions** when rules have exceptions (e.g., P3b with devastating wounds)
6. **Keep test data generators** simple and well-documented
7. **Run full test suite** before committing changes

## Conclusion

This rule-based testing approach provides **comprehensive coverage** without requiring exponential test cases. By validating **mathematical properties** rather than specific scenarios, we ensure the calculator behaves correctly across the entire input space.

The test suite is designed to:
- **Catch regressions** early
- **Document expected behavior** through executable specifications
- **Scale efficiently** as the calculator grows in complexity
- **Provide confidence** in the correctness of probability calculations
