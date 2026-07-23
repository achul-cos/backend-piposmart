# ERD CRM Piposmart

Dokumen ini adalah rancangan target domain CRM Piposmart. Model dibagi menjadi
beberapa bounded context agar diagram tetap terbaca. Nama tabel dan kolom
menggunakan bahasa Inggris supaya konsisten dengan implementasi Go dan SQL.

## 1. Identity, Customer, dan Sales Activity

```mermaid
erDiagram
    direction LR

    ROLES ||--o{ USER_ROLES : grants
    USERS ||--o{ USER_ROLES : receives
    USERS ||--o{ USER_ROLES : assigns

    OWNERS ||--o{ OUTLETS : owns
    OWNERS ||--o{ CUSTOMER_LEADS : has
    OUTLETS o|--o{ CUSTOMER_LEADS : scopes
    USERS o|--o{ CUSTOMER_LEADS : handles_currently

    CUSTOMER_LEADS ||--o{ LEAD_ASSIGNMENTS : assignment_history
    USERS ||--o{ LEAD_ASSIGNMENTS : receives_assignment
    USERS ||--o{ LEAD_ASSIGNMENTS : assigns_lead

    CUSTOMER_LEADS ||--o{ CUSTOMER_INTERACTIONS : records
    USERS ||--o{ CUSTOMER_INTERACTIONS : performs
    REMARK_REASONS ||--o{ CUSTOMER_INTERACTIONS : classifies

    CUSTOMER_LEADS ||--o{ LEAD_STAGE_HISTORIES : changes_stage
    CUSTOMER_INTERACTIONS o|--o{ LEAD_STAGE_HISTORIES : triggers
    USERS ||--o{ LEAD_STAGE_HISTORIES : changes

    CUSTOMER_INTERACTIONS ||--o| TRAINING_REPORTS : may_create
    CUSTOMER_LEADS ||--o{ TRAINING_REPORTS : receives
    USERS ||--o{ TRAINING_REPORTS : conducts

    IMPORT_BATCHES ||--|{ IMPORT_ROWS : contains
    IMPORT_ROWS o|..o{ OWNERS : sources
    IMPORT_ROWS o|..o{ OUTLETS : sources
    IMPORT_ROWS o|..o{ CUSTOMER_LEADS : sources

    ROLES {
        bigint id PK
        varchar code UK
        varchar name
        boolean active
    }

    USERS {
        bigint id PK
        varchar name
        varchar username UK
        varchar email UK
        varchar phone
        varchar password_hash
        varchar account_status
        datetime created_at
        datetime deleted_at
    }

    USER_ROLES {
        bigint user_id PK, FK
        bigint role_id PK, FK
        bigint assigned_by_user_id FK
        datetime assigned_at
    }

    OWNERS {
        bigint id PK
        bigint source_import_row_id FK
        varchar owner_code UK
        varchar name
        varchar email
        varchar phone
        varchar registration_status
        varchar source_type
        datetime created_at
        datetime deleted_at
    }

    OUTLETS {
        bigint id PK
        bigint owner_id FK
        bigint source_import_row_id FK
        varchar outlet_code UK
        varchar brand_name
        varchar outlet_name
        varchar phone
        varchar city
        varchar province
        text address
        datetime deleted_at
    }

    CUSTOMER_LEADS {
        bigint id PK
        bigint owner_id FK
        bigint outlet_id FK
        bigint current_sales_user_id FK
        bigint source_import_row_id FK
        varchar current_stage
        varchar validity_status
        datetime next_follow_up_at
        int total_follow_up
        text latest_note
        datetime created_at
        datetime deleted_at
    }

    LEAD_ASSIGNMENTS {
        bigint id PK
        bigint customer_lead_id FK
        bigint sales_user_id FK
        bigint assigned_by_user_id FK
        datetime assigned_at
        datetime released_at
        varchar release_reason
        varchar status
    }

    REMARK_REASONS {
        bigint id PK
        tinyint score
        varchar code UK
        varchar label
        boolean active
    }

    CUSTOMER_INTERACTIONS {
        bigint id PK
        bigint customer_lead_id FK
        bigint sales_user_id FK
        bigint remark_reason_id FK
        datetime occurred_at
        varchar call_status
        varchar chat_status
        text conclusion
        datetime next_follow_up_at
        int duration_seconds
    }

    LEAD_STAGE_HISTORIES {
        bigint id PK
        bigint customer_lead_id FK
        bigint customer_interaction_id FK
        bigint changed_by_user_id FK
        varchar from_stage
        varchar to_stage
        varchar change_reason
        datetime changed_at
    }

    TRAINING_REPORTS {
        bigint id PK
        bigint customer_lead_id FK
        bigint customer_interaction_id FK
        bigint trainer_user_id FK
        datetime scheduled_at
        datetime held_at
        varchar delivery_mode
        varchar location
        varchar status
        text notes
    }

    IMPORT_BATCHES {
        bigint id PK
        bigint imported_by_user_id FK
        varchar source_type
        varchar file_name
        date period_start
        date period_end
        int row_count
        varchar status
        datetime imported_at
    }

    IMPORT_ROWS {
        bigint id PK
        bigint import_batch_id FK
        int row_number
        json raw_payload
        varchar processing_status
        text error_message
    }
```

## 2. Package, Plan, dan Promotion

```mermaid
erDiagram
    direction LR

    PACKAGES ||--o{ SUBSCRIPTION_PLANS : offers
    PROMOTIONS ||--|{ PROMOTION_OFFERS : contains
    SUBSCRIPTION_PLANS ||--o{ PROMOTION_OFFERS : eligible_plan
    PROMOTION_OFFERS ||--|{ PROMOTION_BENEFITS : grants

    PACKAGES {
        bigint id PK
        varchar code UK
        varchar name
        int tier_rank
        boolean active
        datetime created_at
    }

    SUBSCRIPTION_PLANS {
        bigint id PK
        bigint package_id FK
        varchar code UK
        int tenure_months
        int base_duration_days
        decimal base_price
        char currency
        date effective_from
        date effective_to
        boolean active
    }

    PROMOTIONS {
        bigint id PK
        varchar code UK
        varchar name
        varchar customer_category
        datetime starts_at
        datetime ends_at
        varchar status
        int priority
    }

    PROMOTION_OFFERS {
        bigint id PK
        bigint promotion_id FK
        bigint subscription_plan_id FK
        varchar name
        boolean is_free
        decimal additional_charge
        decimal discount_amount
        int bonus_duration_days
        int priority
        boolean active
    }

    PROMOTION_BENEFITS {
        bigint id PK
        bigint promotion_offer_id FK
        varchar benefit_type
        varchar description
        decimal quantity
        varchar unit
        int duration_days
        varchar product_sku
        decimal estimated_value
    }
```

`PROMOTION_OFFERS` adalah pilihan promo yang dapat dipilih saat closing.
`PROMOTION_BENEFITS` menyimpan komponennya, misalnya tambahan 30 hari,
perangkat POS Android, printer thermal, atau 20 roll kertas thermal.

## 3. Wallet, Subscription, Closing, dan Reconciliation

```mermaid
erDiagram
    direction LR

    OWNERS ||--o{ OUTLETS : owns
    OWNERS ||--o| WALLET_ACCOUNTS : has_wallet
    WALLET_ACCOUNTS ||--o{ PAYMENTS : receives_payment
    WALLET_ACCOUNTS ||--o{ WALLET_TRANSACTIONS : posts
    PAYMENTS ||--o| WALLET_TRANSACTIONS : creates_credit

    OUTLETS ||--o{ SUBSCRIPTION_ORDERS : purchases
    SUBSCRIPTION_PLANS ||--o{ SUBSCRIPTION_ORDERS : selected_plan
    PROMOTION_OFFERS o|--o{ SUBSCRIPTION_ORDERS : selected_offer
    WALLET_TRANSACTIONS ||--o| SUBSCRIPTION_ORDERS : pays

    OUTLETS ||--o{ SUBSCRIPTIONS : subscribes
    SUBSCRIPTIONS ||--o{ SUBSCRIPTION_PERIODS : has_periods
    SUBSCRIPTION_ORDERS ||--o| SUBSCRIPTION_PERIODS : activates
    SUBSCRIPTION_PLANS ||--o{ SUBSCRIPTION_PERIODS : snapshots_plan
    PROMOTION_OFFERS o|--o{ SUBSCRIPTION_PERIODS : applies_offer

    CUSTOMER_LEADS ||--o{ SALES_CLOSINGS : closes
    USERS ||--o{ SALES_CLOSINGS : records
    SUBSCRIPTION_PLANS ||--o{ SALES_CLOSINGS : sells_plan
    PROMOTION_OFFERS o|--o{ SALES_CLOSINGS : sells_offer
    SUBSCRIPTION_ORDERS o|--o| SALES_CLOSINGS : confirms

    SALES_CLOSINGS ||--o{ CLOSING_RECONCILIATIONS : reconciles
    PAYMENTS o|--o{ CLOSING_RECONCILIATIONS : matches_payment
    WALLET_TRANSACTIONS o|--o{ CLOSING_RECONCILIATIONS : matches_wallet
    SUBSCRIPTION_ORDERS o|--o{ CLOSING_RECONCILIATIONS : matches_order
    USERS ||--o{ CLOSING_RECONCILIATIONS : verifies

    OWNERS {
        bigint id PK
        varchar owner_code UK
    }

    OUTLETS {
        bigint id PK
        bigint owner_id FK
        varchar outlet_code UK
    }

    USERS {
        bigint id PK
        varchar name
    }

    CUSTOMER_LEADS {
        bigint id PK
        bigint owner_id FK
        bigint outlet_id FK
    }

    SUBSCRIPTION_PLANS {
        bigint id PK
        bigint package_id FK
        int tenure_months
        int base_duration_days
        decimal base_price
    }

    PROMOTION_OFFERS {
        bigint id PK
        bigint subscription_plan_id FK
        boolean is_free
        decimal additional_charge
        decimal discount_amount
        int bonus_duration_days
    }

    WALLET_ACCOUNTS {
        bigint id PK
        bigint owner_id FK, UK
        varchar external_account_ref UK
        decimal balance
        varchar status
        datetime updated_at
    }

    PAYMENTS {
        bigint id PK
        bigint wallet_account_id FK
        varchar external_payment_ref UK
        varchar payment_method
        decimal gross_amount
        decimal gateway_fee
        decimal settlement_amount
        datetime paid_at
        varchar status
        json raw_payload
    }

    WALLET_TRANSACTIONS {
        bigint id PK
        bigint wallet_account_id FK
        bigint payment_id FK
        varchar external_transaction_ref UK
        varchar transaction_type
        decimal amount
        decimal balance_before
        decimal balance_after
        datetime occurred_at
        varchar status
    }

    SUBSCRIPTION_ORDERS {
        bigint id PK
        bigint outlet_id FK
        bigint subscription_plan_id FK
        bigint promotion_offer_id FK
        bigint wallet_debit_transaction_id FK
        varchar external_order_ref UK
        decimal base_price
        decimal discount_amount
        decimal additional_charge
        decimal total_amount
        datetime purchased_at
        varchar status
    }

    SUBSCRIPTIONS {
        bigint id PK
        bigint outlet_id FK
        varchar external_membership_ref UK
        varchar status
        datetime started_at
        datetime current_end_at
        datetime created_at
    }

    SUBSCRIPTION_PERIODS {
        bigint id PK
        bigint subscription_id FK
        bigint subscription_order_id FK
        bigint subscription_plan_id FK
        bigint promotion_offer_id FK
        datetime starts_at
        int base_duration_days
        int bonus_duration_days
        datetime ends_at
        decimal base_price_snapshot
        json promo_snapshot
        decimal total_paid
    }

    SALES_CLOSINGS {
        bigint id PK
        bigint customer_lead_id FK
        bigint sales_user_id FK
        bigint subscription_plan_id FK
        bigint promotion_offer_id FK
        bigint subscription_order_id FK
        datetime closed_at
        varchar package_snapshot
        int tenure_months_snapshot
        decimal base_price
        decimal discount_amount
        decimal additional_charge
        decimal unique_transfer_code
        decimal final_amount
        varchar status
    }

    CLOSING_RECONCILIATIONS {
        bigint id PK
        bigint sales_closing_id FK
        bigint payment_id FK
        bigint wallet_transaction_id FK
        bigint subscription_order_id FK
        bigint matched_by_user_id FK
        varchar match_type
        varchar status
        datetime matched_at
        text notes
    }
```

## 4. Partner, Referral, Commission, dan Payout

```mermaid
erDiagram
    direction LR

    PARTNER_TYPES ||--o{ PARTNERS : categorizes
    PARTNERS ||--o{ PARTNER_BANK_ACCOUNTS : owns
    PARTNERS ||--o{ PARTNER_ASSIGNMENTS : assignment_history
    USERS ||--o{ PARTNER_ASSIGNMENTS : handles
    USERS ||--o{ PARTNER_ASSIGNMENTS : assigns

    PARTNERS ||--o{ PARTNER_INTERACTIONS : contacted
    USERS ||--o{ PARTNER_INTERACTIONS : performs

    PARTNERS ||--o{ PARTNER_REFERRALS : submits
    CUSTOMER_LEADS ||--o| PARTNER_REFERRALS : originates_from
    USERS ||--o{ PARTNER_REFERRALS : records

    PARTNER_TYPES ||--o{ COMMISSION_RULES : defines
    PACKAGES o|--o{ COMMISSION_RULES : scopes_package
    COMMISSION_RULES ||--o{ COMMISSION_RULE_TIERS : has_tiers

    PARTNER_REFERRALS ||--o| COMMISSION_EARNINGS : earns
    SALES_CLOSINGS ||--o{ COMMISSION_EARNINGS : qualifies
    COMMISSION_RULES ||--o{ COMMISSION_EARNINGS : calculates

    PARTNERS ||--o{ COMMISSION_PAYOUTS : receives
    PARTNER_BANK_ACCOUNTS ||--o{ COMMISSION_PAYOUTS : destination
    USERS ||--o{ COMMISSION_PAYOUTS : approves
    COMMISSION_PAYOUTS ||--|{ COMMISSION_PAYOUT_ITEMS : contains
    COMMISSION_EARNINGS ||--o| COMMISSION_PAYOUT_ITEMS : settles

    USERS {
        bigint id PK
        varchar name
    }

    PACKAGES {
        bigint id PK
        varchar code UK
        varchar name
    }

    CUSTOMER_LEADS {
        bigint id PK
        bigint owner_id FK
        bigint outlet_id FK
    }

    SALES_CLOSINGS {
        bigint id PK
        bigint customer_lead_id FK
        bigint subscription_plan_id FK
        decimal final_amount
        datetime closed_at
    }

    PARTNER_TYPES {
        bigint id PK
        varchar code UK
        varchar name
        boolean active
    }

    PARTNERS {
        bigint id PK
        bigint partner_type_id FK
        varchar partner_code UK
        varchar legal_name
        varchar owner_name
        varchar phone
        varchar email
        varchar status
        date joined_at
    }

    PARTNER_BANK_ACCOUNTS {
        bigint id PK
        bigint partner_id FK
        varchar account_holder
        varchar bank_name
        varchar encrypted_account_number
        boolean primary_account
        boolean active
    }

    PARTNER_ASSIGNMENTS {
        bigint id PK
        bigint partner_id FK
        bigint sales_user_id FK
        bigint assigned_by_user_id FK
        datetime assigned_at
        datetime released_at
        varchar status
    }

    PARTNER_INTERACTIONS {
        bigint id PK
        bigint partner_id FK
        bigint sales_user_id FK
        varchar interaction_type
        datetime occurred_at
        varchar status
        text notes
        datetime next_follow_up_at
    }

    PARTNER_REFERRALS {
        bigint id PK
        bigint partner_id FK
        bigint customer_lead_id FK, UK
        bigint submitted_by_user_id FK
        datetime referred_at
        varchar status
    }

    COMMISSION_RULES {
        bigint id PK
        bigint partner_type_id FK
        bigint package_id FK
        varchar calculation_type
        decimal commission_value
        decimal sales_pic_value
        date effective_from
        date effective_to
        boolean active
    }

    COMMISSION_RULE_TIERS {
        bigint id PK
        bigint commission_rule_id FK
        int min_success_count
        int max_success_count
        decimal bonus_amount
    }

    COMMISSION_EARNINGS {
        bigint id PK
        bigint partner_referral_id FK, UK
        bigint sales_closing_id FK
        bigint commission_rule_id FK
        decimal eligible_amount
        decimal commission_amount
        varchar status
        datetime earned_at
    }

    COMMISSION_PAYOUTS {
        bigint id PK
        bigint partner_id FK
        bigint partner_bank_account_id FK
        bigint approved_by_user_id FK
        datetime requested_at
        datetime approved_at
        datetime paid_at
        decimal total_amount
        varchar status
    }

    COMMISSION_PAYOUT_ITEMS {
        bigint id PK
        bigint commission_payout_id FK
        bigint commission_earning_id FK, UK
        decimal paid_amount
    }
```

## 5. Sales Target dan KPI

```mermaid
erDiagram
    direction LR

    USERS ||--o{ SALES_TARGETS : sets
    SALES_TARGETS ||--|{ SALES_TARGET_ASSIGNMENTS : distributes
    USERS ||--o{ SALES_TARGET_ASSIGNMENTS : receives

    KPI_DEFINITIONS ||--o{ KPI_RESULTS : measures
    USERS ||--o{ KPI_RESULTS : evaluated_sales
    SALES_TARGET_ASSIGNMENTS o|--o{ KPI_RESULTS : compares_target

    USERS {
        bigint id PK
        varchar name
    }

    SALES_TARGETS {
        bigint id PK
        bigint set_by_user_id FK
        date period_month UK
        varchar name
        varchar scope
        int default_customer_target
        decimal default_revenue_target
        varchar status
        datetime created_at
    }

    SALES_TARGET_ASSIGNMENTS {
        bigint id PK
        bigint sales_target_id FK
        bigint sales_user_id FK
        int customer_target
        decimal revenue_target
        date effective_from
        date effective_to
    }

    KPI_DEFINITIONS {
        bigint id PK
        varchar code UK
        varchar name
        varchar metric_type
        decimal weight
        json thresholds
        date effective_from
        date effective_to
        boolean active
    }

    KPI_RESULTS {
        bigint id PK
        bigint kpi_definition_id FK
        bigint sales_user_id FK
        bigint sales_target_assignment_id FK
        date period_start
        date period_end
        decimal target_value
        decimal actual_value
        decimal score
        varchar rating
        datetime generated_at
    }
```

## Aturan dan constraint utama

1. Role resmi adalah `ADMIN`, `SUPERVISOR`, dan `SALES`. Hak akses tidak
   ditentukan dari nama user, tetapi dari `USER_ROLES`.
2. Satu owner dapat memiliki banyak outlet. Lead dan subscription dapat
   diarahkan ke outlet tertentu.
3. Hanya boleh ada satu `LEAD_ASSIGNMENTS` berstatus `ACTIVE` untuk satu lead.
   Riwayat assignment tidak ditimpa.
4. Remark adalah hasil interaksi, sedangkan `current_stage` adalah state lead.
   Remark 1 tidak boleh menurunkan stage `POTENTIAL` menjadi `POSSIBLE`.
5. Remark 0 melepaskan assignment aktif. Lead tetap dapat didistribusikan ulang.
6. Masa aktif paket menggunakan hari tetap:
   `base_duration_days = tenure_months * 30` dan
   `ends_at = starts_at + base_duration_days + bonus_duration_days`.
7. Promo gratis dipilih lebih dahulu berdasarkan `is_free` dan `priority`.
   Promo berbayar tetap merupakan pilihan eksplisit pengguna.
8. Semua nilai uang menggunakan `DECIMAL`, bukan `FLOAT`.
9. Top-up/payment adalah kejadian omzet. Closing adalah kejadian performa
   penjualan. Keduanya dipertemukan melalui reconciliation agar tidak dihitung
   ganda.
10. Harga, paket, tenor, dan promo disimpan sebagai snapshot pada closing dan
    subscription period agar laporan historis tidak berubah saat master diubah.
11. Satu commission earning berasal dari satu referral/customer yang closing.
    Payout selalu memiliki payout item sehingga sumber komisi dapat ditelusuri.
12. Dashboard dan KPI membaca source-of-truth transaksi/aktivitas. Nilai agregat
    sebaiknya berupa query/view atau snapshot terjadwal, bukan diinput manual.

