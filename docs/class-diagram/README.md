# Class Diagram CRM Piposmart

Class diagram ini menggambarkan model domain dan perilaku aplikasi target.
Diagram sengaja tidak dibuat sebagai salinan satu-banding-satu tabel database:
entity menjaga state, policy menjaga aturan bisnis, dan service mengorkestrasi
beberapa aggregate/repository.

## 1. Customer dan Sales Activity

```mermaid
classDiagram
    class User {
        +uint64 ID
        +string Name
        +AccountStatus Status
        +hasRole(roleCode) bool
        +can(permission) bool
    }

    class Role {
        +uint64 ID
        +RoleCode Code
        +string Name
    }

    class Owner {
        +uint64 ID
        +string OwnerCode
        +string Name
        +string Phone
        +addOutlet(outlet)
    }

    class Outlet {
        +uint64 ID
        +uint64 OwnerID
        +string BrandName
        +string OutletName
        +string Phone
    }

    class CustomerLead {
        +uint64 ID
        +uint64 OwnerID
        +uint64 OutletID
        +LeadStage CurrentStage
        +ValidityStatus Validity
        +datetime NextFollowUpAt
        +int TotalFollowUp
        +assignTo(salesID)
        +release(reason)
        +applyStage(nextStage)
        +scheduleFollowUp(at)
    }

    class LeadAssignment {
        +uint64 ID
        +uint64 LeadID
        +uint64 SalesUserID
        +datetime AssignedAt
        +datetime ReleasedAt
        +AssignmentStatus Status
        +release(reason)
        +isActive() bool
    }

    class RemarkReason {
        +uint64 ID
        +int Score
        +string Code
        +string Label
    }

    class CustomerInteraction {
        +uint64 ID
        +uint64 LeadID
        +uint64 SalesUserID
        +CallStatus CallStatus
        +ChatStatus ChatStatus
        +RemarkReason Remark
        +string Conclusion
        +datetime OccurredAt
        +datetime NextFollowUpAt
    }

    class LeadStageHistory {
        +uint64 ID
        +LeadStage From
        +LeadStage To
        +string Reason
        +datetime ChangedAt
    }

    class TrainingReport {
        +uint64 ID
        +uint64 LeadID
        +datetime ScheduledAt
        +datetime HeldAt
        +TrainingMode Mode
        +TrainingStatus Status
        +string Notes
        +markCompleted(heldAt)
        +reschedule(at)
    }

    class RoleCode {
        <<enumeration>>
        ADMIN
        SUPERVISOR
        SALES
    }

    class LeadStage {
        <<enumeration>>
        NEW
        INVALID
        POSSIBLE
        POTENTIAL
        SUBSCRIBED
    }

    class CallStatus {
        <<enumeration>>
        NO_CALL
        CONTACTED
        CONNECTED
        ENGAGE
        INTEREST
        PROSPECT
        UNINTEREST
    }

    class ChatStatus {
        <<enumeration>>
        NO_CHAT
        SENT
        DELIVERED
        ENGAGE
        INTEREST
        PROSPECT
        UNINTEREST
    }

    User "*" -- "*" Role : has
    Owner "1" *-- "1..*" Outlet : owns
    Owner "1" *-- "0..*" CustomerLead : has
    Outlet "0..1" -- "0..*" CustomerLead : scopes
    CustomerLead "1" *-- "0..*" LeadAssignment : assignmentHistory
    User "1" -- "0..*" LeadAssignment : sales
    CustomerLead "1" *-- "0..*" CustomerInteraction : interactions
    User "1" -- "0..*" CustomerInteraction : performs
    RemarkReason "1" -- "0..*" CustomerInteraction : classifies
    CustomerLead "1" *-- "0..*" LeadStageHistory : stageHistory
    CustomerInteraction "0..1" -- "0..*" LeadStageHistory : triggers
    CustomerInteraction "1" *-- "0..1" TrainingReport : creates
```

## 2. Package, Promo, Wallet, Subscription, dan Closing

```mermaid
classDiagram
    class Package {
        +uint64 ID
        +string Code
        +string Name
        +int TierRank
    }

    class SubscriptionPlan {
        +uint64 ID
        +Package Package
        +int TenureMonths
        +int BaseDurationDays
        +Money BasePrice
        +bool Active
        +calculateBaseEnd(startAt) datetime
    }

    class Promotion {
        +uint64 ID
        +string Code
        +string Name
        +CustomerCategory Category
        +datetime StartsAt
        +datetime EndsAt
        +PromotionStatus Status
        +isActive(at) bool
    }

    class PromotionOffer {
        +uint64 ID
        +uint64 PlanID
        +bool IsFree
        +Money AdditionalCharge
        +Money DiscountAmount
        +int BonusDurationDays
        +int Priority
        +calculateFinalPrice(basePrice) Money
    }

    class PromotionBenefit {
        +uint64 ID
        +BenefitType Type
        +string Description
        +decimal Quantity
        +string Unit
        +int DurationDays
        +string ProductSKU
    }

    class WalletAccount {
        +uint64 ID
        +uint64 OwnerID
        +Money Balance
        +credit(amount)
        +debit(amount)
        +hasSufficientBalance(amount) bool
    }

    class Payment {
        +uint64 ID
        +string ExternalReference
        +PaymentMethod Method
        +Money GrossAmount
        +Money GatewayFee
        +Money SettlementAmount
        +datetime PaidAt
        +markSettled()
    }

    class WalletTransaction {
        +uint64 ID
        +WalletTransactionType Type
        +Money Amount
        +Money BalanceBefore
        +Money BalanceAfter
        +datetime OccurredAt
    }

    class SubscriptionOrder {
        +uint64 ID
        +uint64 OutletID
        +uint64 PlanID
        +uint64 PromotionOfferID
        +Money BasePrice
        +Money Discount
        +Money AdditionalCharge
        +Money Total
        +datetime PurchasedAt
        +confirm()
        +cancel()
    }

    class Subscription {
        +uint64 ID
        +uint64 OutletID
        +SubscriptionStatus Status
        +datetime CurrentEndAt
        +activate(period)
        +renew(period)
        +expire(at)
    }

    class SubscriptionPeriod {
        +uint64 ID
        +datetime StartsAt
        +int BaseDurationDays
        +int BonusDurationDays
        +datetime EndsAt
        +Money TotalPaid
        +PlanSnapshot PlanSnapshot
        +PromoSnapshot PromoSnapshot
    }

    class SalesClosing {
        +uint64 ID
        +uint64 LeadID
        +uint64 SalesUserID
        +PlanSnapshot PlanSnapshot
        +PromoSnapshot PromoSnapshot
        +Money FinalAmount
        +datetime ClosedAt
        +ClosingStatus Status
        +linkOrder(orderID)
    }

    class ClosingReconciliation {
        +uint64 ID
        +uint64 ClosingID
        +uint64 PaymentID
        +uint64 WalletTransactionID
        +uint64 SubscriptionOrderID
        +ReconciliationStatus Status
        +match()
        +reject(reason)
    }

    class BenefitType {
        <<enumeration>>
        FREE_DURATION
        DISCOUNT
        DEVICE
        CONSUMABLE
        OTHER
    }

    class WalletTransactionType {
        <<enumeration>>
        TOPUP
        SUBSCRIPTION_DEBIT
        ADJUSTMENT
        REFUND
    }

    class SubscriptionStatus {
        <<enumeration>>
        PENDING
        ACTIVE
        EXPIRED
        CANCELLED
    }

    Package "1" *-- "1..*" SubscriptionPlan : plans
    Promotion "1" *-- "1..*" PromotionOffer : offers
    SubscriptionPlan "1" -- "0..*" PromotionOffer : eligiblePlan
    PromotionOffer "1" *-- "1..*" PromotionBenefit : benefits
    WalletAccount "1" *-- "0..*" WalletTransaction : ledger
    Payment "0..1" -- "0..1" WalletTransaction : createsTopup
    SubscriptionOrder "1" -- "1" WalletTransaction : paidBy
    SubscriptionPlan "1" -- "0..*" SubscriptionOrder : selectedPlan
    PromotionOffer "0..1" -- "0..*" SubscriptionOrder : selectedOffer
    Subscription "1" *-- "1..*" SubscriptionPeriod : periods
    SubscriptionOrder "1" -- "0..1" SubscriptionPeriod : activates
    SalesClosing "1" *-- "0..*" ClosingReconciliation : reconciliation
    SubscriptionOrder "0..1" -- "0..1" SalesClosing : confirms
```

## 3. Partner, Commission, Target, dan KPI

```mermaid
classDiagram
    class PartnerType {
        +uint64 ID
        +string Code
        +string Name
    }

    class Partner {
        +uint64 ID
        +string PartnerCode
        +string LegalName
        +PartnerStatus Status
        +addReferral(leadID)
        +changePIC(salesID)
    }

    class PartnerAssignment {
        +uint64 ID
        +uint64 PartnerID
        +uint64 SalesUserID
        +datetime AssignedAt
        +datetime ReleasedAt
        +release()
    }

    class PartnerInteraction {
        +uint64 ID
        +uint64 PartnerID
        +uint64 SalesUserID
        +InteractionType Type
        +datetime OccurredAt
        +string Notes
        +datetime NextFollowUpAt
    }

    class PartnerReferral {
        +uint64 ID
        +uint64 PartnerID
        +uint64 CustomerLeadID
        +ReferralStatus Status
        +datetime ReferredAt
        +markQualified()
        +markClosed()
    }

    class CommissionRule {
        +uint64 ID
        +uint64 PartnerTypeID
        +uint64 PackageID
        +CommissionCalculationType Type
        +decimal CommissionValue
        +date EffectiveFrom
        +calculate(eligibleAmount) Money
    }

    class CommissionRuleTier {
        +uint64 ID
        +int MinSuccessCount
        +int MaxSuccessCount
        +Money BonusAmount
        +matches(successCount) bool
    }

    class CommissionEarning {
        +uint64 ID
        +uint64 ReferralID
        +uint64 ClosingID
        +Money EligibleAmount
        +Money CommissionAmount
        +EarningStatus Status
        +markPayable()
        +markPaid()
    }

    class CommissionPayout {
        +uint64 ID
        +uint64 PartnerID
        +Money TotalAmount
        +PayoutStatus Status
        +request()
        +approve(adminID)
        +markPaid(at)
    }

    class CommissionPayoutItem {
        +uint64 ID
        +uint64 EarningID
        +Money PaidAmount
    }

    class SalesTarget {
        +uint64 ID
        +date PeriodMonth
        +TargetScope Scope
        +int DefaultCustomerTarget
        +Money DefaultRevenueTarget
        +publish()
    }

    class SalesTargetAssignment {
        +uint64 ID
        +uint64 SalesUserID
        +int CustomerTarget
        +Money RevenueTarget
        +overrideTargets(customer, revenue)
    }

    class KPIDefinition {
        +uint64 ID
        +string Code
        +string MetricType
        +decimal Weight
        +Thresholds Thresholds
        +evaluate(target, actual) KPIEvaluation
    }

    class KPIResult {
        +uint64 ID
        +uint64 SalesUserID
        +date PeriodStart
        +date PeriodEnd
        +decimal TargetValue
        +decimal ActualValue
        +decimal Score
        +string Rating
    }

    PartnerType "1" *-- "0..*" Partner : categorizes
    Partner "1" *-- "0..*" PartnerAssignment : assignmentHistory
    Partner "1" *-- "0..*" PartnerInteraction : interactions
    Partner "1" *-- "0..*" PartnerReferral : referrals
    PartnerType "1" -- "0..*" CommissionRule : rules
    CommissionRule "1" *-- "0..*" CommissionRuleTier : tiers
    PartnerReferral "1" -- "0..1" CommissionEarning : earns
    CommissionRule "1" -- "0..*" CommissionEarning : calculates
    CommissionPayout "1" *-- "1..*" CommissionPayoutItem : items
    CommissionEarning "1" -- "0..1" CommissionPayoutItem : settledBy
    SalesTarget "1" *-- "1..*" SalesTargetAssignment : assignments
    KPIDefinition "1" -- "0..*" KPIResult : evaluates
    SalesTargetAssignment "0..1" -- "0..*" KPIResult : targetSource
```

## 4. Domain Service dan Policy

```mermaid
classDiagram
    class LeadAssignmentService {
        +assignLead(leadID, salesID, actorID)
        +releaseLead(leadID, reason, actorID)
        +redistributeLead(leadID, nextSalesID, supervisorID)
    }

    class CustomerStagePolicy {
        +resolveNextStage(currentStage, remarkScore) LeadStage
        +shouldReleasePIC(remarkScore) bool
        +canDowngrade(currentStage, nextStage) bool
    }

    class InteractionService {
        +recordInteraction(command) CustomerInteraction
        +scheduleTraining(command) TrainingReport
    }

    class PromoSelectionService {
        +findEligibleOffers(planID, category, at) PromotionOffer[]
        +prioritizeFree(offers) PromotionOffer[]
        +selectOffer(offerID) PromotionOffer
    }

    class SubscriptionDurationPolicy {
        +baseDays(tenureMonths) int
        +calculateEndDate(startAt, baseDays, bonusDays) datetime
    }

    class WalletService {
        +recordTopup(payment) WalletTransaction
        +debitForOrder(order) WalletTransaction
        +adjustBalance(command) WalletTransaction
    }

    class ClosingService {
        +recordClosing(command) SalesClosing
        +createOrderFromClosing(closingID) SubscriptionOrder
    }

    class ReconciliationService {
        +matchTopup(closingID, paymentID)
        +matchPurchase(closingID, orderID)
        +findHangingTransactions(period) ReconciliationIssue[]
    }

    class CommissionService {
        +calculateForReferral(referralID, closingID) CommissionEarning
        +createPayout(partnerID, earningIDs) CommissionPayout
        +approvePayout(payoutID, adminID)
    }

    class KPIService {
        +calculateSalesMetrics(salesID, period) SalesMetrics
        +evaluateKPI(salesID, period) KPIResult[]
        +rankSales(period) SalesRanking[]
    }

    class LeadRepository {
        <<interface>>
        +findByID(id) CustomerLead
        +save(lead)
        +findUnassigned(filter) CustomerLead[]
    }

    class WalletRepository {
        <<interface>>
        +findByOwner(ownerID) WalletAccount
        +save(account)
        +appendTransaction(transaction)
    }

    class SubscriptionRepository {
        <<interface>>
        +findActiveByOutlet(outletID) Subscription
        +save(subscription)
        +saveOrder(order)
    }

    class PartnerRepository {
        <<interface>>
        +findByID(id) Partner
        +save(partner)
        +findPayableEarnings(partnerID) CommissionEarning[]
    }

    LeadAssignmentService ..> LeadRepository
    LeadAssignmentService ..> CustomerStagePolicy
    InteractionService ..> LeadRepository
    InteractionService ..> CustomerStagePolicy
    PromoSelectionService ..> SubscriptionDurationPolicy
    WalletService ..> WalletRepository
    ClosingService ..> PromoSelectionService
    ClosingService ..> SubscriptionRepository
    ReconciliationService ..> WalletRepository
    ReconciliationService ..> SubscriptionRepository
    CommissionService ..> PartnerRepository
    KPIService ..> LeadRepository
    KPIService ..> SubscriptionRepository
```

## Invariant domain penting

- `CustomerStagePolicy` menjaga agar remark 1 tidak menurunkan customer dari
  `POTENTIAL` ke `POSSIBLE`.
- Remark 0 menghasilkan stage `INVALID` dan memicu pelepasan assignment aktif.
- `PromoSelectionService` mengurutkan promo gratis lebih dahulu; promo berbayar
  tidak boleh terpilih otomatis tanpa konfirmasi.
- `SubscriptionDurationPolicy` selalu menggunakan `tenureMonths * 30`, bukan
  penambahan bulan kalender.
- `WalletAccount` hanya berubah melalui ledger `WalletTransaction`.
- Closing tidak otomatis dianggap top-up baru. `ReconciliationService`
  menghubungkan closing dengan payment, wallet debit, dan subscription order.
- `CommissionPayout` tidak boleh dibayar tanpa item earning yang menunjukkan
  referral/customer sumber komisinya.
- KPI dihitung dari aktivitas dan closing yang sudah tersimpan, bukan angka yang
  diketik ulang oleh Sales.

