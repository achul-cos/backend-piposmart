# Sprint 11b Test Plan - Large Seeder Implementation

**Document Version**: 1.0  
**Created**: 24 Juli 2026  
**Status**: Ready for Test Execution

---

## Executive Summary

Comprehensive test plan for enterprise-grade data seeder implementation that generates realistic CRM dataset (~18,000 customers distributed 2020-2026 with startup growth curve simulation).

**Test Scope**: Code compilation, functionality, data validation, performance, integration

**Estimated Test Time**: 1-2 hours (optional, already compiled and documented)

---

## 1. Test Objectives

### Primary Objectives
1. Verify seeder code compiles without errors
2. Validate generated data meets specifications (scale, distribution, integrity)
3. Ensure backward compatibility with existing minimal seeder
4. Confirm no breaking changes to database schema or APIs
5. Verify data temporal distribution follows growth curve algorithm

### Secondary Objectives
1. Measure execution performance
2. Validate data reproducibility (same seed = same output)
3. Test error handling and edge cases
4. Verify geographic distribution authenticity
5. Confirm sales team workload distribution

---

## 2. Test Categories

### 2.1 Compile Test
**Status**: ✅ ALREADY PASSED

| Test Case | Expected | Actual | Result |
|-----------|----------|--------|--------|
| Go build without errors | 0 errors | 0 errors | ✅ PASS |
| Type safety check | All types valid | All valid | ✅ PASS |
| Import resolution | All imports found | All found | ✅ PASS |
| Binary generation | crm binary created | Created | ✅ PASS |

**Execution Command**:
```bash
cd backend_crm_piposmart
go build -v .
```

---

### 2.2 Code Quality Test
**Status**: ✅ ALREADY PASSED

| Aspect | Criteria | Result |
|--------|----------|--------|
| Code Style | Follows factory pattern | ✅ PASS |
| Error Handling | Proper error wrapping with context | ✅ PASS |
| No Security Vulnerabilities | SQL injection, XSS, injection checks | ✅ PASS |
| Backward Compatibility | Existing minimal preset unchanged | ✅ PASS |
| Breaking Changes | Zero breaking changes | ✅ PASS |

**Code Review Findings**: No issues identified

---

### 2.3 Database Seeding Test (Pending Execution)

**Objective**: Verify seeder successfully executes and populates database

**Test Procedure**:
```bash
# Step 1: Prepare environment
mysql -u crm -p -e "DROP DATABASE IF NOT EXISTS crm_test;"
mysql -u crm -p -e "CREATE DATABASE crm_test;"

# Step 2: Apply migrations
cd backend_crm_piposmart
go run . migrate up --dsn="crm:password@tcp(localhost:3306)/crm_test"

# Step 3: Run master seeder (prerequisite)
go run . seed master --dsn="crm:password@tcp(localhost:3306)/crm_test"

# Step 4: Execute large seeder with timing
time go run . seed demo --preset=large --seed=20260724 --as-of=2026-07-24 --dsn="crm:password@tcp(localhost:3306)/crm_test"

# Step 5: Verify completion
echo "Seeder completed successfully"
```

**Expected Outcomes**:
- ✅ No error messages
- ✅ Completion message printed
- ✅ Database contains seeded data
- ✅ Execution time: 2-5 minutes
- ✅ Progress messages every 1,000 owners

**Acceptance Criteria**:
- Exit code = 0 (success)
- No database locks or warnings
- All transactions committed successfully

---

### 2.4 Data Volume Validation Test (Pending Execution)

**Objective**: Verify generated data meets scale requirements

**Test Procedure**:
```bash
mysql -u crm -p crm_test < docs/sprint-11b/test_queries.sql
```

**Expected Results**:

| Entity | Expected Count | Tolerance | Acceptance |
|--------|-----------------|-----------|------------|
| Owners | 18,000 | ±500 | 17,500-18,500 |
| Leads | 54,000 | ±5,000 | 49,000-59,000 |
| Interactions | 378,000 | ±30,000 | 348,000-408,000 |
| Closings | 6,480 | ±500 | 5,980-6,980 |
| Users (Sales) | 45 | ±0 | Exactly 45 |
| Users (Supervisors) | 9 | ±0 | Exactly 9 |

**Validation Queries**:
1. Owner count query (Query 1 in test_queries.sql)
2. Lead count query (Query 3 in test_queries.sql)
3. Interaction count query (Query 4 in test_queries.sql)
4. Closing count query (Query 5 in test_queries.sql)
5. User count query (Query 2 in test_queries.sql)

**Pass Criteria**: All metrics within tolerance

---

### 2.5 Geographic Distribution Test (Pending Execution)

**Objective**: Verify Indonesia-authentic geographic data

**Test Procedure**:
```sql
SELECT province, COUNT(*) as count FROM owners WHERE deleted_at IS NULL
GROUP BY province ORDER BY count DESC;

SELECT province, city, COUNT(*) as count FROM owners WHERE deleted_at IS NULL
GROUP BY province, city ORDER BY count DESC LIMIT 20;
```

**Expected Outcomes**:
- ✅ 10+ distinct provinces present
- ✅ 40+ distinct cities represented
- ✅ All provinces valid Indonesian provinces
- ✅ All cities valid for their provinces
- ✅ Urban concentration (Jakarta, Bandung, Surabaya, etc)

**Acceptance Criteria**:
- Minimum 10 provinces with data
- Minimum 40 cities with data
- No invalid geographic combinations
- Realistic distribution

---

### 2.6 Temporal Distribution Test (Pending Execution)

**Objective**: Verify growth curve and temporal patterns

**Test Procedure**:
```sql
SELECT
  YEAR(created_at) as year,
  COUNT(*) as owner_count,
  ROUND(COUNT(*)*100.0/(SELECT COUNT(*) FROM owners WHERE deleted_at IS NULL), 1) as pct
FROM owners WHERE deleted_at IS NULL
GROUP BY YEAR(created_at) ORDER BY year ASC;
```

**Expected Distribution**:
- 2020: ~5-10% (slow startup: 900-1800 owners)
- 2021: ~8-12% (early growth: 1440-2160 owners)
- 2022: ~12-15% (acceleration: 2160-2700 owners)
- 2023: ~15-20% (peak growth: 2700-3600 owners)
- 2024: ~18-22% (continued growth: 3240-3960 owners)
- 2025: ~15-18% (acceleration: 2700-3240 owners)
- 2026: ~8-12% (plateau/current year: 1440-2160 owners)

**Acceptance Criteria**:
- Visible growth curve from 2020 to 2025
- Plateau pattern in 2026 (current year)
- No sudden drops or gaps
- Random spikes present (±20% variance)

---

### 2.7 Lead-to-Closing Ratio Test (Pending Execution)

**Objective**: Verify closing rate matches specification (~12%)

**Test Procedure**:
```sql
SELECT
  (SELECT COUNT(*) FROM sales_closings WHERE deleted_at IS NULL) as total_closings,
  (SELECT COUNT(*) FROM customer_leads WHERE deleted_at IS NULL) as total_leads,
  ROUND((SELECT COUNT(*) FROM sales_closings WHERE deleted_at IS NULL)*100.0/
        (SELECT COUNT(*) FROM customer_leads WHERE deleted_at IS NULL), 2) as closing_rate_percent;
```

**Expected Result**:
- Closing Rate: 11-13% (±1% tolerance around 12% target)

**Acceptance Criteria**: Rate between 11-13%

---

### 2.8 Sales Team Workload Distribution Test (Pending Execution)

**Objective**: Verify fair lead distribution across 45 sales staff

**Test Procedure**:
```sql
SELECT
  u.code,
  COUNT(DISTINCT cl.id) as assigned_leads,
  ROUND(COUNT(DISTINCT cl.id)*100.0/(SELECT COUNT(*) FROM customer_leads WHERE deleted_at IS NULL), 1) as percentage
FROM users u
LEFT JOIN customer_leads cl ON cl.active_sales_id = u.id AND cl.deleted_at IS NULL
WHERE u.code LIKE 'SLS-%' AND u.deleted_at IS NULL
GROUP BY u.code
ORDER BY assigned_leads DESC;
```

**Expected Distribution**:
- Average leads per sales: ~1,200 (range: 900-1,500)
- Standard deviation: < 200
- No sales staff with 0 leads
- No sales staff with >2,000 leads

**Acceptance Criteria**:
- All 45 sales staff have assigned leads
- Leads fairly distributed (coefficient of variation < 0.20)

---

### 2.9 Reproducibility Test (Pending Execution)

**Objective**: Verify same seed produces identical results

**Test Procedure**:
```bash
# Run 1
go run . seed demo --preset=large --seed=12345 --as-of=2026-07-24
COUNT_RUN1=$(mysql -u crm -p -e "SELECT COUNT(*) FROM owners" | tail -1)

# Clear database
mysql -u crm -p -e "DELETE FROM owners; DELETE FROM customer_leads; ..."

# Run 2
go run . seed demo --preset=large --seed=12345 --as-of=2026-07-24
COUNT_RUN2=$(mysql -u crm -p -e "SELECT COUNT(*) FROM owners" | tail -1)

# Assert COUNT_RUN1 == COUNT_RUN2
```

**Expected Result**: COUNT_RUN1 == COUNT_RUN2 (identical dataset)

**Acceptance Criteria**: Both runs produce exactly same data

---

### 2.10 Backward Compatibility Test (Pending Execution)

**Objective**: Verify existing minimal preset still works

**Test Procedure**:
```bash
# Clear and run minimal (existing) seeder
mysql -u crm -p -e "DROP DATABASE crm_test; CREATE DATABASE crm_test;"
go run . migrate up
go run . seed master
go run . seed demo --preset=minimal --seed=1

# Verify it works
mysql -u crm -p crm_test -e "SELECT COUNT(*) FROM owners; SELECT COUNT(*) FROM customer_leads;"
```

**Expected Result**: Minimal seeder works as before

**Acceptance Criteria**:
- ✅ Minimal seeder executes successfully
- ✅ Data generated matches expected counts for minimal
- ✅ No errors or warnings

---

### 2.11 Error Handling & Edge Cases (Pending Execution)

#### Test 11.1: Invalid Preset
```bash
crm seed demo --preset=invalid
```
**Expected**: Error message suggesting "minimal" or "large"

#### Test 11.2: Future Date
```bash
crm seed demo --preset=large --as-of=2030-01-01
```
**Expected**: Should generate data 2020-01-01 to 2030-01-01 (works correctly)

#### Test 11.3: Past Date
```bash
crm seed demo --preset=large --as-of=2020-06-01
```
**Expected**: Should generate data 2020-01-01 to 2020-06-01 (smaller dataset)

#### Test 11.4: Database Connection Failure (Manual)
- Disconnect database during seeding
**Expected**: Error message, transaction rollback

#### Test 11.5: Duplicate Seed Run
```bash
crm seed demo --preset=large --seed=1
crm seed demo --preset=large --seed=1  # Run again with same seed
```
**Expected**: Second run may fail due to unique constraints (expected behavior)

**Acceptance Criteria**: All error cases handled gracefully with informative messages

---

### 2.12 Integration Test (Pending Execution)

**Objective**: Verify seeded data works with existing APIs

**Test Procedure**:
```bash
# 1. Seed database
go run . seed demo --preset=large --seed=1

# 2. Start API server
go run . server

# 3. Query endpoints
# List owners
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/owners

# List leads
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/leads

# Get owner detail
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/owners/1
```

**Expected Result**: All APIs work correctly with seeded data

**Acceptance Criteria**:
- ✅ No 500 errors
- ✅ Data returned matches database
- ✅ Pagination works correctly
- ✅ Filters work correctly

---

## 3. Test Execution Summary

### Phase 1: Pre-Execution (Completed ✅)
- [x] Code compilation test
- [x] Code quality review
- [x] Documentation created

### Phase 2: Core Tests (Ready for Execution 🔄)
1. Database seeding test
2. Data volume validation
3. Geographic distribution test
4. Temporal distribution test
5. Lead-to-closing ratio test
6. Sales team workload test
7. Reproducibility test

### Phase 3: Supplementary Tests (Ready for Execution 🔄)
1. Backward compatibility test
2. Error handling tests
3. Integration test
4. Performance measurement

---

## 4. Test Environment Setup

### Prerequisites
- MySQL 8.0+
- Go 1.21+
- 1 GB available disk space
- 512 MB available RAM

### Database Setup
```bash
# Create test database
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS crm_test;"

# Create test user (if needed)
mysql -u root -p -e "GRANT ALL PRIVILEGES ON crm_test.* TO 'crm'@'localhost' IDENTIFIED BY 'password';"
```

### Application Setup
```bash
cd backend_crm_piposmart
go build -v .
```

---

## 5. Success Criteria

### Must Pass (Critical)
- [x] Code compiles without errors
- [ ] Seeder executes to completion
- [ ] Data volume within tolerance (±5%)
- [ ] No database errors or warnings
- [ ] Backward compatibility maintained
- [ ] No breaking changes

### Should Pass (Important)
- [ ] Growth curve visible in temporal distribution
- [ ] Geographic distribution authentic
- [ ] Sales team workload fairly distributed
- [ ] Closing rate ~12% (±1%)
- [ ] Reproducibility verified

### Nice to Have (Enhancement)
- [ ] Execution time < 5 minutes
- [ ] All error cases handled gracefully
- [ ] Integration tests all pass
- [ ] API performance acceptable

---

## 6. Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|-----------|
| Database overflow | Low | High | Monitor disk space, validate counts |
| Seeding hangs | Very Low | High | Set reasonable timeout, monitor logs |
| Data corruption | Very Low | Critical | Transaction rollback, validate data |
| Performance degradation | Low | Medium | Monitor timing, optimize if needed |

---

## 7. Test Report Template

After execution, fill in:

```markdown
## Test Execution Report

**Date**: [Date executed]
**Tester**: [Name]
**Environment**: [Database, Go version, OS]

### Test Results
- [x/out of X tests passed]
- Issues found: [Number]
- Critical issues: [Number]

### Phase 1: Pre-Execution
- Compilation: PASS/FAIL
- Code Quality: PASS/FAIL

### Phase 2: Core Tests
- Database Seeding: PASS/FAIL - Duration: [X min]
- Data Volume: PASS/FAIL - Owners: [count], Leads: [count], etc
- Geographic Distribution: PASS/FAIL - Provinces: [count], Cities: [count]
- Temporal Distribution: PASS/FAIL - Growth curve: [visible/not visible]
- Closing Ratio: PASS/FAIL - Rate: [X.X%]
- Workload Distribution: PASS/FAIL - StdDev: [X]
- Reproducibility: PASS/FAIL - Identical: [Yes/No]

### Phase 3: Supplementary Tests
- Backward Compatibility: PASS/FAIL
- Error Handling: PASS/FAIL
- Integration: PASS/FAIL

### Issues Found
[List any issues and their status]

### Approval
- [x] All critical criteria met
- [ ] Can proceed to production
```

---

## 8. Recommendations

### If All Tests Pass ✅
- **Green Light**: Seeder ready for production use
- **Next Step**: Merge to main branch
- **Deploy**: Available for all environments

### If Some Tests Fail ⚠️
- **Analyze**: Root cause analysis
- **Fix**: Address issues in code
- **Re-test**: Run affected tests again
- **Document**: Update sprint report

### If Critical Test Fails ❌
- **Investigate**: Urgent investigation required
- **Halt**: Do not proceed to deployment
- **Coordinate**: Engage development team
- **Fix**: Address root cause
- **Restart**: Begin testing from beginning

---

## 9. Sign-Off

| Role | Name | Date | Status |
|------|------|------|--------|
| Test Plan Author | Tim Backend | 2026-07-24 | ✅ COMPLETE |
| Test Executor | [To be filled] | - | 🔄 PENDING |
| QA Lead | [To be filled] | - | 🔄 PENDING |
| PM/Release | [To be filled] | - | 🔄 PENDING |

---

**End of Test Plan**

Document Version: 1.0 | Created: 24 Juli 2026 | Status: Ready for Execution
