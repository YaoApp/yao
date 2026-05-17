import { log, Process } from "@yao/runtime";

function Init(force: boolean = false) {
  if (!force && IsInstalled()) {
    log.Info(
      "Application is already initialized. Use 'yao run scripts.setup.Init true' to force reinit."
    );
    return { status: "skipped", message: "Already initialized" };
  }

  log.Info("=== Starting Agent Test Application Init ===");

  log.Info("Running database migration...");
  MigrateModels();

  SetupRoles();
  SetupTypes();
  SetupMenus();
  SetupInvitationCodes();

  const rootInfo = SetupRootUser();

  log.Info("=== Application Init Completed ===");
  console.log("");
  console.log("========================================");
  console.log("  Root User Credentials");
  console.log("----------------------------------------");
  console.log(`  Email:    ${ROOT_USER.email}`);
  console.log(`  Password: ${ROOT_USER_PASSWORD}`);
  if (rootInfo) {
    console.log(`  UserID:   ${rootInfo.userId}`);
    console.log(`  TeamID:   ${rootInfo.teamId}`);
  }
  console.log("========================================");

  return { status: "success", message: "Initialization completed" };
}

function IsInstalled(): boolean {
  try {
    const users = Process("models.__yao.user.Get", {
      wheres: [{ column: "email", value: ROOT_USER.email }],
      limit: 1,
    });
    if (users && users.length > 0) {
      return true;
    }
    const roles = Process("models.__yao.role.Get", { limit: 1 });
    if (roles && roles.length > 0) {
      return true;
    }
    return false;
  } catch (e) {
    return false;
  }
}

function MigrateModels() {
  const models = ["menu"];
  for (const model of models) {
    try {
      Process(`models.${model}.Migrate`, false);
      log.Info(`Migrated model: ${model}`);
    } catch (e) {
      log.Warn(`Failed to migrate model ${model}: ${e}`);
    }
  }
}

// ============================================
// Seed Data Import
// ============================================

function SetupRoles() {
  log.Info("Setting up roles...");
  const result = Process("seeds.import", "roles.csv", "__yao.role", {
    chunk_size: 100,
    duplicate: "ignore",
    mode: "each",
  });
  log.Info(
    `Roles: Total=${result.total}, Success=${result.success}, Ignored=${result.ignore}, Failed=${result.failure}`
  );
}

function SetupTypes() {
  log.Info("Setting up user types...");
  const result = Process("seeds.import", "types.csv", "__yao.user.type", {
    chunk_size: 100,
    duplicate: "ignore",
    mode: "each",
  });
  log.Info(
    `Types: Total=${result.total}, Success=${result.success}, Ignored=${result.ignore}, Failed=${result.failure}`
  );
}

function SetupMenus() {
  log.Info("Setting up menus...");
  const result = Process("seeds.import", "menus.csv", "menu", {
    chunk_size: 100,
    duplicate: "ignore",
    mode: "each",
  });
  log.Info(
    `Menus: Total=${result.total}, Success=${result.success}, Ignored=${result.ignore}, Failed=${result.failure}`
  );
}

function SetupInvitationCodes() {
  log.Info("Setting up invitation codes...");
  const result = Process(
    "seeds.import",
    "invitation_codes.csv",
    "__yao.invitation",
    { chunk_size: 100, duplicate: "ignore", mode: "each" }
  );
  log.Info(
    `Invitation codes: Total=${result.total}, Success=${result.success}, Ignored=${result.ignore}, Failed=${result.failure}`
  );
}

// ============================================
// ID Generation
// ============================================

function generateId(): string {
  const timestamp = Date.now().toString().slice(-8);
  const random = Math.floor(Math.random() * 10000)
    .toString()
    .padStart(4, "0");
  return timestamp + random;
}

// ============================================
// Root User
// ============================================

const ROOT_USER_PASSWORD = "Yao123++";

const ROOT_USER = {
  email: "root@yaoagents.com",
  name: "Administrator",
  status: "active",
  role_id: "system:root",
  type: "selfhosting",
  locale: "en-us",
};

const ROOT_TEAM = {
  name: "Administrator's Team",
  display_name: "Administrator's Team",
  description: "Default team for root user",
  status: "active",
  type: "other",
  is_verified: true,
  role_id: "system:root",
};

function SetupRootUser(): { userId: string; teamId: string } | null {
  log.Info("Setting up root user...");

  const existing = Process("models.__yao.user.Get", {
    wheres: [{ column: "email", value: ROOT_USER.email }],
    limit: 1,
  });

  if (existing && existing.length > 0) {
    log.Info(`Root user already exists: ${ROOT_USER.email}`);
    const userId = existing[0].user_id;
    const teams = Process("models.__yao.team.Get", {
      wheres: [{ column: "owner_id", value: userId }],
      limit: 1,
    });
    const teamId = teams && teams.length > 0 ? teams[0].team_id : null;
    return teamId ? { userId, teamId } : null;
  }

  const userId = generateId();
  const teamId = generateId();
  const memberId = generateId();

  Process("models.__yao.user.Save", {
    ...ROOT_USER,
    user_id: userId,
    password_hash: ROOT_USER_PASSWORD,
  });
  log.Info(`Root user created: ${ROOT_USER.email} (user_id: ${userId})`);

  Process("models.__yao.team.Save", {
    ...ROOT_TEAM,
    team_id: teamId,
    owner_id: userId,
    contact_email: ROOT_USER.email,
  });
  log.Info(`Root team created (team_id: ${teamId})`);

  Process("models.__yao.member.Save", {
    member_id: memberId,
    team_id: teamId,
    user_id: userId,
    member_type: "user",
    display_name: ROOT_USER.name,
    email: ROOT_USER.email,
    role_id: "team:owner",
    is_owner: true,
    status: "active",
    joined_at: new Date().toISOString(),
  });
  log.Info(`Root member created (member_id: ${memberId})`);

  return { userId, teamId };
}

// ============================================
// Test Users
// ============================================

const TEST_PASSWORD = "Test123++";

interface TestUser {
  email: string;
  name: string;
  role_id: string;
  teamRole: string;
  is_owner: boolean;
}

const ALPHA_TEAM = {
  name: "Alpha Team",
  display_name: "Alpha Team",
  description: "Test team A for permission isolation tests",
  status: "active",
  type: "other",
  is_verified: true,
  role_id: "user:*",
};

const ALPHA_USERS: TestUser[] = [
  {
    email: "alpha-owner@test.local",
    name: "Alpha Owner",
    role_id: "user:*",
    teamRole: "team:owner",
    is_owner: true,
  },
  {
    email: "alpha-admin@test.local",
    name: "Alpha Admin",
    role_id: "user:*",
    teamRole: "team:admin",
    is_owner: false,
  },
  {
    email: "alpha-member@test.local",
    name: "Alpha Member",
    role_id: "user:*",
    teamRole: "team:member",
    is_owner: false,
  },
];

const BETA_TEAM = {
  name: "Beta Team",
  display_name: "Beta Team",
  description: "Test team B for permission isolation tests",
  status: "active",
  type: "other",
  is_verified: true,
  role_id: "user:*",
};

const BETA_USERS: TestUser[] = [
  {
    email: "beta-owner@test.local",
    name: "Beta Owner",
    role_id: "user:*",
    teamRole: "team:owner",
    is_owner: true,
  },
  {
    email: "beta-member@test.local",
    name: "Beta Member",
    role_id: "user:*",
    teamRole: "team:member",
    is_owner: false,
  },
];

function createTeamWithUsers(
  teamDef: Record<string, any>,
  users: TestUser[],
  ownerUser: TestUser
): { teamId: string; userIds: Record<string, string> } {
  const teamId = generateId();
  const ownerId = generateId();

  Process("models.__yao.user.Save", {
    email: ownerUser.email,
    name: ownerUser.name,
    status: "active",
    role_id: ownerUser.role_id,
    type: "free",
    locale: "en-us",
    user_id: ownerId,
    password_hash: TEST_PASSWORD,
  });

  Process("models.__yao.team.Save", {
    ...teamDef,
    team_id: teamId,
    owner_id: ownerId,
    contact_email: ownerUser.email,
  });

  Process("models.__yao.member.Save", {
    member_id: generateId(),
    team_id: teamId,
    user_id: ownerId,
    member_type: "user",
    display_name: ownerUser.name,
    email: ownerUser.email,
    role_id: ownerUser.teamRole,
    is_owner: true,
    status: "active",
    joined_at: new Date().toISOString(),
  });

  const userIds: Record<string, string> = {};
  userIds[ownerUser.email] = ownerId;

  for (const u of users) {
    if (u.email === ownerUser.email) continue;
    const uid = generateId();
    Process("models.__yao.user.Save", {
      email: u.email,
      name: u.name,
      status: "active",
      role_id: u.role_id,
      type: "free",
      locale: "en-us",
      user_id: uid,
      password_hash: TEST_PASSWORD,
    });
    Process("models.__yao.member.Save", {
      member_id: generateId(),
      team_id: teamId,
      user_id: uid,
      member_type: "user",
      display_name: u.name,
      email: u.email,
      role_id: u.teamRole,
      is_owner: u.is_owner,
      status: "active",
      joined_at: new Date().toISOString(),
    });
    userIds[u.email] = uid;
  }

  return { teamId, userIds };
}

/**
 * SetupTestUsers - Create test user matrix for unit tests
 * yao run scripts.setup.SetupTestUsers
 */
function SetupTestUsers() {
  log.Info("=== Setting up test users ===");

  const existing = Process("models.__yao.user.Get", {
    wheres: [{ column: "email", value: "alpha-owner@test.local" }],
    limit: 1,
  });
  if (existing && existing.length > 0) {
    log.Info("Test users already exist, skipping.");
    return;
  }

  const alphaOwner = ALPHA_USERS.find((u) => u.is_owner)!;
  const alpha = createTeamWithUsers(ALPHA_TEAM, ALPHA_USERS, alphaOwner);
  log.Info(`Alpha Team created: team_id=${alpha.teamId}`);

  const betaOwner = BETA_USERS.find((u) => u.is_owner)!;
  const beta = createTeamWithUsers(BETA_TEAM, BETA_USERS, betaOwner);
  log.Info(`Beta Team created: team_id=${beta.teamId}`);

  const rootUsers = Process("models.__yao.user.Get", {
    wheres: [{ column: "email", value: ROOT_USER.email }],
    limit: 1,
  });
  const rootUserId = rootUsers && rootUsers.length > 0 ? rootUsers[0].user_id : "";
  const rootTeams = Process("models.__yao.team.Get", {
    wheres: [{ column: "owner_id", value: rootUserId }],
    limit: 1,
  });
  const rootTeamId = rootTeams && rootTeams.length > 0 ? rootTeams[0].team_id : "";

  const envContent = [
    `TEST_ROOT_USER_ID=${rootUserId}`,
    `TEST_ROOT_TEAM_ID=${rootTeamId}`,
    `TEST_ALPHA_TEAM_ID=${alpha.teamId}`,
    `TEST_ALPHA_OWNER_USER_ID=${alpha.userIds["alpha-owner@test.local"]}`,
    `TEST_ALPHA_ADMIN_USER_ID=${alpha.userIds["alpha-admin@test.local"]}`,
    `TEST_ALPHA_MEMBER_USER_ID=${alpha.userIds["alpha-member@test.local"]}`,
    `TEST_BETA_TEAM_ID=${beta.teamId}`,
    `TEST_BETA_OWNER_USER_ID=${beta.userIds["beta-owner@test.local"]}`,
    `TEST_BETA_MEMBER_USER_ID=${beta.userIds["beta-member@test.local"]}`,
    "",
  ].join("\n");

  console.log("__TEST_USERS_ENV_BEGIN__");
  console.log(envContent);
  console.log("__TEST_USERS_ENV_END__");

  log.Info("=== Test users setup completed ===");
}
