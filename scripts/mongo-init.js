// =========================================================================
// MONGODB LOCAL DEVELOPMENT SEED
// =========================================================================

// Ensure this matches the database name in your connection string
db = db.getSiblingDB('aigateway');

print("Starting MongoDB Database Seeding...");

// =========================================================================
// Setup Indexes (Enforce Data Integrity)
// =========================================================================
print("Configuring Indexes...");

// Prevent multiple tenants from having the same API key
db.identities.createIndex({ api_key_hash: 1 }, { unique: true });

// Prevent duplicate Casbin rules from being created
db.casbin_rule.createIndex(
  { ptype: 1, v0: 1, v1: 1, v2: 1 }, 
  { unique: true }
);

// =========================================================================
// Seed AuthN Identities (Idempotent Upsert)
// =========================================================================
print("Seeding Identities...");

db.identities.updateOne(
  { api_key_hash: "7ba88f380e2bd2b3a94f74a0f3cdb2b502fdf251f829ad44cd009de9f632ed96" },
  {
    $set: {
      tenant_id: "tenant-local-dev",
      tier: "premium"
    }
  },
  { upsert: true } // Creates it if missing, updates if it already exists
);

print("Identities seeded successfully.");

// =========================================================================
// Seed AuthZ Casbin Policies (Idempotent Upsert)
// =========================================================================
print("Seeding Casbin Policies...");

const devPolicies = [
  { ptype: "p", v0: "tenant-local-dev", v1: "gpt-4o", v2: "use" },
  { ptype: "p", v0: "tenant-local-dev", v1: "gemini-1.5-pro", v2: "use" }
];

devPolicies.forEach(policy => {
  db.casbin_rule.updateOne(
    { 
      ptype: policy.ptype, 
      v0: policy.v0, 
      v1: policy.v1, 
      v2: policy.v2 
    },
    { $set: policy },
    { upsert: true }
  );
});

print("Casbin Policies seeded successfully.");
print("Database Seeding Completed!");
