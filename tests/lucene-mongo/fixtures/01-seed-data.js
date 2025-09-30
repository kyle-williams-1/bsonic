// Seed data for BSON integration testing
// This script runs when MongoDB container starts for the first time

db = db.getSiblingDB('bsonic_test');

// Create collections and insert test data
print('Seeding BSON integration test data...');

// Users collection - basic user data with various field types
db.users.insertMany([
  {
    _id: ObjectId(),
    name: "John Doe",
    email: "john.doe@example.com",
    age: 30,
    active: true,
    role: "admin",
    tags: ["developer", "golang", "mongodb"],
    profile: {
      bio: "Senior software engineer",
      location: "San Francisco, CA",
      website: "https://johndoe.dev"
    },
    created_at: new Date("2023-01-15T10:30:00Z"),
    last_login: new Date("2024-01-10T14:22:00Z")
  },
  {
    _id: ObjectId(),
    name: "Jane Smith",
    email: "jane.smith@example.com",
    age: 28,
    active: true,
    role: "user",
    tags: ["designer", "ui", "ux"],
    profile: {
      bio: "UX/UI Designer",
      location: "New York, NY",
      website: "https://janesmith.design"
    },
    created_at: new Date("2023-02-20T09:15:00Z"),
    last_login: new Date("2024-01-12T16:45:00Z")
  },
  {
    _id: ObjectId(),
    name: "Bob Johnson",
    email: "bob.johnson@example.com",
    age: 35,
    active: false,
    role: "user",
    tags: ["manager", "leadership"],
    profile: {
      bio: "Project Manager",
      location: "Chicago, IL",
      website: null
    },
    created_at: new Date("2022-11-10T14:20:00Z"),
    last_login: new Date("2023-12-15T11:30:00Z")
  },
  {
    _id: ObjectId(),
    name: "Alice Brown",
    email: "alice.brown@example.com",
    age: 25,
    active: true,
    role: "moderator",
    tags: ["content", "writing", "blog"],
    profile: {
      bio: "Content Writer",
      location: "Austin, TX",
      website: "https://alicebrown.blog"
    },
    created_at: new Date("2023-06-05T13:45:00Z"),
    last_login: new Date("2024-01-14T08:20:00Z")
  },
  {
    _id: ObjectId(),
    name: "Charlie Wilson",
    email: "charlie.wilson@example.com",
    age: 42,
    active: true,
    role: "admin",
    tags: ["devops", "kubernetes", "docker"],
    profile: {
      bio: "DevOps Engineer",
      location: "Seattle, WA",
      website: "https://charliewilson.tech"
    },
    created_at: new Date("2022-08-30T16:10:00Z"),
    last_login: new Date("2024-01-13T12:15:00Z")
  }
]);

// Products collection - e-commerce style data
db.products.insertMany([
  {
    _id: ObjectId(),
    name: "Wireless Headphones",
    category: "electronics",
    price: 99.99,
    in_stock: true,
    tags: ["audio", "wireless", "bluetooth"],
    specifications: {
      battery_life: "30 hours",
      connectivity: "Bluetooth 5.0",
      weight: "250g"
    },
    reviews: [
      { user_id: ObjectId(), rating: 5, comment: "Great sound quality!" },
      { user_id: ObjectId(), rating: 4, comment: "Good value for money" }
    ],
    created_at: new Date("2023-10-15T10:00:00Z"),
    updated_at: new Date("2024-01-05T14:30:00Z")
  },
  {
    _id: ObjectId(),
    name: "Gaming Mouse",
    category: "electronics",
    price: 79.99,
    in_stock: true,
    tags: ["gaming", "mouse", "rgb"],
    specifications: {
      dpi: "16000",
      connectivity: "USB",
      weight: "120g"
    },
    reviews: [
      { user_id: ObjectId(), rating: 5, comment: "Perfect for gaming" }
    ],
    created_at: new Date("2023-11-20T09:30:00Z"),
    updated_at: new Date("2024-01-08T11:45:00Z")
  },
  {
    _id: ObjectId(),
    name: "Coffee Mug",
    category: "home",
    price: 15.99,
    in_stock: false,
    tags: ["kitchen", "ceramic", "coffee"],
    specifications: {
      material: "ceramic",
      capacity: "12oz",
      dishwasher_safe: true
    },
    reviews: [],
    created_at: new Date("2023-12-01T08:00:00Z"),
    updated_at: new Date("2023-12-15T16:20:00Z")
  }
]);

// Orders collection - complex nested data
db.orders.insertMany([
  {
    _id: ObjectId(),
    order_number: "ORD-001",
    customer: {
      name: "John Doe",
      email: "john.doe@example.com",
      address: {
        street: "123 Main St",
        city: "San Francisco",
        state: "CA",
        zip: "94102"
      }
    },
    items: [
      {
        product_id: ObjectId(),
        name: "Wireless Headphones",
        quantity: 1,
        price: 99.99
      }
    ],
    total: 99.99,
    status: "completed",
    payment_method: "credit_card",
    created_at: new Date("2024-01-10T10:30:00Z"),
    shipped_at: new Date("2024-01-11T14:00:00Z")
  },
  {
    _id: ObjectId(),
    order_number: "ORD-002",
    customer: {
      name: "Jane Smith",
      email: "jane.smith@example.com",
      address: {
        street: "456 Oak Ave",
        city: "New York",
        state: "NY",
        zip: "10001"
      }
    },
    items: [
      {
        product_id: ObjectId(),
        name: "Gaming Mouse",
        quantity: 2,
        price: 79.99
      }
    ],
    total: 159.98,
    status: "pending",
    payment_method: "paypal",
    created_at: new Date("2024-01-12T16:45:00Z"),
    shipped_at: null
  }
]);

// Create indexes for better query performance
db.users.createIndex({ "name": 1 });
db.users.createIndex({ "email": 1 });
db.users.createIndex({ "role": 1 });
db.users.createIndex({ "active": 1 });
db.users.createIndex({ "tags": 1 });
db.users.createIndex({ "profile.location": 1 });

// Create text indexes for full-text search
// Text indexes must be created on all fields you want to be searchable
db.users.createIndex({ 
  "name": "text", 
  "email": "text", 
  "profile.bio": "text",
  "tags": "text"
});

db.products.createIndex({ "name": 1 });
db.products.createIndex({ "category": 1 });
db.products.createIndex({ "price": 1 });
db.products.createIndex({ "in_stock": 1 });
db.products.createIndex({ "tags": 1 });

// Create text index for products
db.products.createIndex({ 
  "name": "text", 
  "category": "text",
  "tags": "text",
  "specifications.battery_life": "text",
  "specifications.connectivity": "text",
  "specifications.material": "text"
});

db.orders.createIndex({ "order_number": 1 });
db.orders.createIndex({ "customer.email": 1 });
db.orders.createIndex({ "status": 1 });
db.orders.createIndex({ "created_at": 1 });

// Create text index for orders
db.orders.createIndex({ 
  "order_number": "text",
  "customer.name": "text",
  "customer.email": "text",
  "status": "text",
  "payment_method": "text"
});

print('BSON integration test data seeded successfully!');
print('Collections created: users, products, orders');
print('Total documents inserted: ' + (db.users.countDocuments() + db.products.countDocuments() + db.orders.countDocuments()));
