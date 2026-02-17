#!/usr/bin/env node
/**
 * OpenOMS Discord Server Setup Script
 *
 * Usage:
 *   1. Create a Discord bot at https://discord.com/developers/applications
 *   2. Enable "Server Members Intent" and "Message Content Intent" in bot settings
 *   3. Invite bot to your server with Administrator permissions:
 *      https://discord.com/oauth2/authorize?client_id=YOUR_APP_ID&permissions=8&scope=bot
 *   4. Run:
 *      DISCORD_BOT_TOKEN=your_token DISCORD_GUILD_ID=your_server_id npm run setup
 *
 * The script is idempotent — running it again won't create duplicates.
 */

import {
  Client,
  GatewayIntentBits,
  ChannelType,
  PermissionFlagsBits,
  Colors,
  GuildVerificationLevel,
  GuildExplicitContentFilter,
  GuildDefaultMessageNotifications,
  AutoModerationRuleEventType,
  AutoModerationRuleTriggerType,
  AutoModerationActionType,
} from "discord.js";

const TOKEN = process.env.DISCORD_BOT_TOKEN;
const GUILD_ID = process.env.DISCORD_GUILD_ID;

if (!TOKEN || !GUILD_ID) {
  console.error(
    "Missing env vars. Usage:\n  DISCORD_BOT_TOKEN=xxx DISCORD_GUILD_ID=yyy npm run setup"
  );
  process.exit(1);
}

// ── Configuration ────────────────────────────────────────────────

const ROLES = [
  {
    name: "Maintainer",
    color: Colors.Red,
    hoist: true,
    mentionable: true,
    reason: "Core team / project maintainers",
  },
  {
    name: "Contributor",
    color: Colors.Green,
    hoist: true,
    mentionable: true,
    reason: "People who contributed code or docs",
  },
  {
    name: "Uzytkownik",
    color: Colors.Blue,
    hoist: false,
    mentionable: false,
    reason: "Regular community members / users",
  },
];

const CATEGORIES = [
  {
    name: "INFORMACJE",
    channels: [
      {
        name: "ogloszenia",
        topic:
          "Oficjalne ogłoszenia projektu OpenOMS. Tylko maintainerzy mogą pisać.",
        readOnly: true,
      },
      {
        name: "roadmap",
        topic:
          "Aktualny roadmap i postępy prac. https://github.com/openoms-org/openoms/blob/main/ROADMAP.md",
        readOnly: true,
      },
      {
        name: "changelog",
        topic:
          "Logi zmian z każdego release'u. Automatyczne powiadomienia z GitHub Actions.",
        readOnly: true,
      },
    ],
  },
  {
    name: "SPOLECZNOSC",
    channels: [
      {
        name: "general",
        topic:
          "Ogólne rozmowy o OpenOMS, e-commerce i zarządzaniu zamówieniami.",
      },
      {
        name: "pokaz-swoje",
        topic:
          "Pochwal się swoją instalacją OpenOMS, customizacjami, integracjami.",
      },
      {
        name: "pomysly",
        topic:
          "Propozycje nowych funkcji i usprawnień. Głosuj reakcjami!",
      },
    ],
  },
  {
    name: "POMOC",
    channels: [
      {
        name: "instalacja",
        topic:
          "Pomoc z instalacją: Docker, Go, Node.js, PostgreSQL, konfiguracja .env",
        slowMode: 10,
      },
      {
        name: "konfiguracja",
        topic:
          "Ustawienia tenanta, SMTP, CORS, JWT, szyfrowanie, RBAC, 2FA",
        slowMode: 10,
      },
      {
        name: "integracje",
        topic:
          "Allegro, InPost, DHL, marketplace'y, feedy IOF, API webhooks",
        slowMode: 10,
      },
    ],
  },
  {
    name: "ROZWOJ",
    channels: [
      {
        name: "contributing",
        topic:
          "Dyskusje o kodzie, architekturze, PR review. CONTRIBUTING.md: https://github.com/openoms-org/openoms/blob/main/CONTRIBUTING.md",
      },
      {
        name: "bugs",
        topic:
          "Zgłaszanie i dyskusja o bugach. Dla oficjalnych zgłoszeń użyj GitHub Issues.",
      },
      {
        name: "pull-requests",
        topic: "Dyskusje o otwartych PR-ach i code review.",
      },
    ],
  },
  {
    name: "BOTY",
    channels: [
      {
        name: "github-feed",
        topic:
          "Automatyczne powiadomienia: nowe release'y, issues, pull requesty z GitHub.",
        readOnly: true,
      },
    ],
  },
];

const WELCOME_MESSAGE = `# Witaj na serwerze OpenOMS!

Open-source Order Management System dla polskiego e-commerce.

## Przydatne linki
- **GitHub:** https://github.com/openoms-org/openoms
- **Strona:** https://openoms.org
- **Dokumentacja:** https://github.com/openoms-org/openoms/tree/main/docs
- **Zgłoś bug:** https://github.com/openoms-org/openoms/issues/new

## Kanały
- **#general** — ogólne rozmowy
- **#instalacja** — pomoc z setupem
- **#integracje** — Allegro, InPost, marketplace'y
- **#contributing** — chcesz pomóc? Zacznij tutaj!
- **#pomysly** — zaproponuj nową funkcję

## Role
- **@Maintainer** — core team
- **@Contributor** — kontrybutorzy kodu
- **@Uzytkownik** — użytkownicy OpenOMS

## Zasady
1. Bądź kulturalny — szanuj innych
2. Pisz po polsku lub angielsku
3. Nie spamuj, nie reklamuj
4. Pytania techniczne → odpowiedni kanał w POMOC
5. Bugi → GitHub Issues (nie tutaj)

Licencja: AGPLv3 (core) + MIT (SDK packages)
`;

// ── AutoMod rules ────────────────────────────────────────────────

const AUTOMOD_RULES = [
  {
    name: "[OpenOMS] Blokada spamu",
    eventType: AutoModerationRuleEventType.MessageSend,
    triggerType: AutoModerationRuleTriggerType.Keyword,
    triggerMetadata: {
      keywordFilter: [
        // Crypto/NFT scams
        "free nitro",
        "free discord nitro",
        "steam gift",
        "airdrop*",
        "crypto giveaway",
        "nft drop",
        "claim your reward",
        // Phishing patterns
        "discord.gg.com*",
        "discordgift*",
        "discorad*",
        "dlscord*",
        // Adult/NSFW spam
        "onlyfans.com*",
        "18+ content",
        // Generic spam triggers
        "dm me for*",
        "earn money fast",
        "work from home easy",
        "make $* per day",
      ],
    },
    actions: [
      {
        type: AutoModerationActionType.BlockMessage,
        metadata: {
          customMessage:
            "Wiadomość zablokowana przez AutoMod. Jeśli to pomyłka, skontaktuj się z @Maintainer.",
        },
      },
      {
        type: AutoModerationActionType.SendAlertMessage,
        metadata: { channelId: null }, // set dynamically to #github-feed or first mod channel
      },
    ],
  },
  {
    name: "[OpenOMS] Ochrona przed mass mention",
    eventType: AutoModerationRuleEventType.MessageSend,
    triggerType: AutoModerationRuleTriggerType.MentionSpam,
    triggerMetadata: {
      mentionTotalLimit: 5,
    },
    actions: [
      {
        type: AutoModerationActionType.BlockMessage,
        metadata: {
          customMessage:
            "Zbyt wiele wzmianek w jednej wiadomości. Limit: 5.",
        },
      },
      {
        type: AutoModerationActionType.Timeout,
        metadata: { durationSeconds: 300 }, // 5 min timeout
      },
    ],
  },
  {
    name: "[OpenOMS] Blokada zaprosze Discord",
    eventType: AutoModerationRuleEventType.MessageSend,
    triggerType: AutoModerationRuleTriggerType.Keyword,
    triggerMetadata: {
      regexPatterns: [
        "(?:https?://)?(?:www\\.)?discord(?:\\.gg|app\\.com/invite)/[a-zA-Z0-9]+",
      ],
    },
    actions: [
      {
        type: AutoModerationActionType.BlockMessage,
        metadata: {
          customMessage:
            "Linki z zaproszeniami do innych serwerów Discord nie są dozwolone.",
        },
      },
    ],
  },
];

// ── Script ───────────────────────────────────────────────────────

const client = new Client({
  intents: [
    GatewayIntentBits.Guilds,
    GatewayIntentBits.GuildMembers,
    GatewayIntentBits.AutoModerationConfiguration,
  ],
});

async function findOrCreate(collection, name, createFn) {
  const existing = collection.cache.find(
    (item) => item.name.toLowerCase() === name.toLowerCase()
  );
  if (existing) {
    console.log(`  [skip] "${name}" already exists`);
    return existing;
  }
  const created = await createFn();
  console.log(`  [created] "${name}"`);
  return created;
}

async function run() {
  await client.login(TOKEN);
  const guild = await client.guilds.fetch(GUILD_ID);
  console.log(
    `Connected to: ${guild.name} (${guild.memberCount} members)\n`
  );

  // ── 1. Server-level security settings ──
  console.log("=== SECURITY SETTINGS ===");

  // Verification level: Medium — must be registered on Discord for 5+ min
  if (guild.verificationLevel !== GuildVerificationLevel.Medium) {
    await guild.setVerificationLevel(GuildVerificationLevel.Medium);
    console.log(
      "  [set] Verification level -> Medium (registered 5+ min)"
    );
  } else {
    console.log("  [skip] Verification level already Medium");
  }

  // Content filter: scan messages from all members (not just no-role)
  if (
    guild.explicitContentFilter !==
    GuildExplicitContentFilter.AllMembers
  ) {
    await guild.setExplicitContentFilter(
      GuildExplicitContentFilter.AllMembers
    );
    console.log(
      "  [set] Explicit content filter -> All members"
    );
  } else {
    console.log("  [skip] Content filter already on All members");
  }

  // Default notifications: only @mentions (no spam from every message)
  if (
    guild.defaultMessageNotifications !==
    GuildDefaultMessageNotifications.OnlyMentions
  ) {
    await guild.setDefaultMessageNotifications(
      GuildDefaultMessageNotifications.OnlyMentions
    );
    console.log(
      "  [set] Default notifications -> Only @mentions"
    );
  } else {
    console.log("  [skip] Notifications already Only @mentions");
  }

  // 2FA requirement for moderator actions
  try {
    if (guild.mfaLevel === 0) {
      await guild.setMFALevel(1);
      console.log(
        "  [set] 2FA required for moderators"
      );
    } else {
      console.log("  [skip] 2FA for moderators already enabled");
    }
  } catch {
    console.log(
      "  [skip] 2FA for moderators (requires server owner to have 2FA enabled)"
    );
  }

  // ── 2. Roles ──
  console.log("\n=== ROLES ===");
  await guild.roles.fetch();
  const roleMap = {};

  for (const roleDef of ROLES) {
    const role = await findOrCreate(guild.roles, roleDef.name, () =>
      guild.roles.create({
        name: roleDef.name,
        color: roleDef.color,
        hoist: roleDef.hoist,
        mentionable: roleDef.mentionable,
        reason: roleDef.reason,
      })
    );
    roleMap[roleDef.name] = role;
  }

  const maintainerRole = roleMap["Maintainer"];
  const everyoneRole = guild.roles.everyone;

  // ── 3. Harden @everyone permissions ──
  console.log("\n=== @EVERYONE PERMISSIONS ===");
  const denyEveryone = [
    PermissionFlagsBits.MentionEveryone, // no @everyone/@here
    PermissionFlagsBits.ManageMessages, // no message management
    PermissionFlagsBits.ManageChannels, // no channel management
    PermissionFlagsBits.ManageRoles, // no role management
    PermissionFlagsBits.ManageGuild, // no server settings
    PermissionFlagsBits.Administrator, // obviously
    PermissionFlagsBits.BanMembers, // no bans
    PermissionFlagsBits.KickMembers, // no kicks
    PermissionFlagsBits.ManageWebhooks, // no webhook tampering
    PermissionFlagsBits.ManageNicknames, // no nickname changing of others
    PermissionFlagsBits.CreateInstantInvite, // no invite creation (maintainers only)
  ];

  const currentDenied = everyoneRole.permissions;
  const needsUpdate = denyEveryone.some(
    (perm) => currentDenied.has(perm)
  );
  if (needsUpdate || true) {
    // always apply to be safe
    const newPerms = everyoneRole.permissions;
    for (const perm of denyEveryone) {
      newPerms.remove(perm);
    }
    await everyoneRole.setPermissions(newPerms);
    console.log(
      "  [set] @everyone: removed dangerous permissions (mention everyone, manage channels/roles, create invites, kick/ban)"
    );
  }

  // ── 4. Categories & Channels ──
  console.log("\n=== CHANNELS ===");
  await guild.channels.fetch();

  for (const catDef of CATEGORIES) {
    console.log(`\n[category] ${catDef.name}`);

    const category = await findOrCreate(
      guild.channels,
      catDef.name,
      () =>
        guild.channels.create({
          name: catDef.name,
          type: ChannelType.GuildCategory,
        })
    );

    for (const chDef of catDef.channels) {
      const channel = await findOrCreate(
        guild.channels,
        chDef.name,
        () =>
          guild.channels.create({
            name: chDef.name,
            type: ChannelType.GuildText,
            parent: category.id,
            topic: chDef.topic,
            rateLimitPerUser: chDef.slowMode || 0,
          })
      );

      // Move to correct category if it exists but is in wrong category
      if (channel.parentId !== category.id) {
        await channel.setParent(category.id, {
          lockPermissions: false,
        });
        console.log(
          `  [moved] "${chDef.name}" -> ${catDef.name}`
        );
      }

      // Set topic if missing
      if (chDef.topic && channel.topic !== chDef.topic) {
        await channel.setTopic(chDef.topic);
      }

      // Set slow mode if configured
      if (
        chDef.slowMode &&
        channel.rateLimitPerUser !== chDef.slowMode
      ) {
        await channel.setRateLimitPerUser(chDef.slowMode);
        console.log(
          `  [perms] "${chDef.name}" -> slow mode ${chDef.slowMode}s`
        );
      }

      // Read-only channels: only Maintainers can send
      if (chDef.readOnly) {
        await channel.permissionOverwrites.set([
          {
            id: everyoneRole.id,
            deny: [PermissionFlagsBits.SendMessages],
            allow: [PermissionFlagsBits.ViewChannel],
          },
          {
            id: maintainerRole.id,
            allow: [
              PermissionFlagsBits.SendMessages,
              PermissionFlagsBits.ManageMessages,
            ],
          },
        ]);
        console.log(
          `  [perms] "${chDef.name}" -> read-only (Maintainer can post)`
        );
      }
    }
  }

  // ── 5. AutoMod rules ──
  console.log("\n=== AUTOMOD RULES ===");

  // Find alert channel (use github-feed or first available)
  const alertChannel = guild.channels.cache.find(
    (c) => c.name === "github-feed"
  );

  const existingRules = await guild.autoModerationRules.fetch();

  for (const ruleDef of AUTOMOD_RULES) {
    const existing = existingRules.find(
      (r) => r.name === ruleDef.name
    );
    if (existing) {
      console.log(`  [skip] "${ruleDef.name}" already exists`);
      continue;
    }

    // Set alert channel on SendAlertMessage actions
    const actions = ruleDef.actions
      .map((a) => {
        if (
          a.type === AutoModerationActionType.SendAlertMessage &&
          alertChannel
        ) {
          return {
            ...a,
            metadata: { ...a.metadata, channelId: alertChannel.id },
          };
        }
        if (
          a.type === AutoModerationActionType.SendAlertMessage &&
          !alertChannel
        ) {
          return null; // skip alert if no channel
        }
        return a;
      })
      .filter(Boolean);

    try {
      await guild.autoModerationRules.create({
        name: ruleDef.name,
        enabled: true,
        eventType: ruleDef.eventType,
        triggerType: ruleDef.triggerType,
        triggerMetadata: ruleDef.triggerMetadata,
        actions,
        exemptRoles: [maintainerRole.id], // Maintainers bypass AutoMod
      });
      console.log(`  [created] "${ruleDef.name}"`);
    } catch (err) {
      console.log(
        `  [error] "${ruleDef.name}": ${err.message}`
      );
    }
  }

  // ── 6. Welcome message in #ogloszenia ──
  console.log("\n=== WELCOME MESSAGE ===");
  const announceChannel = guild.channels.cache.find(
    (c) => c.name === "ogloszenia"
  );
  if (announceChannel) {
    const messages = await announceChannel.messages.fetch({
      limit: 5,
    });
    const hasWelcome = messages.some((m) =>
      m.content.includes("Witaj na serwerze OpenOMS")
    );
    if (!hasWelcome) {
      await announceChannel.send(WELCOME_MESSAGE);
      console.log("  [sent] Welcome message in #ogloszenia");
    } else {
      console.log("  [skip] Welcome message already exists");
    }
  }

  // ── 7. Server settings ──
  console.log("\n=== SERVER SETTINGS ===");
  const generalChannel = guild.channels.cache.find(
    (c) => c.name === "general"
  );

  // Set system channel to #general
  if (
    generalChannel &&
    guild.systemChannelId !== generalChannel.id
  ) {
    await guild.setSystemChannel(generalChannel.id);
    console.log("  [set] System channel -> #general");
  }

  // Set server description
  const description =
    "OpenOMS — open-source Order Management System dla polskiego e-commerce. AGPLv3 + MIT.";
  if (guild.description !== description) {
    try {
      await guild.setDescription(description);
      console.log("  [set] Server description");
    } catch {
      console.log(
        "  [skip] Server description (requires Community or boost level)"
      );
    }
  }

  // ── Summary ──
  console.log("\n╔══════════════════════════════════════════╗");
  console.log("║       SETUP COMPLETE                     ║");
  console.log("╚══════════════════════════════════════════╝\n");

  console.log("Security applied:");
  console.log(
    "  - Verification level: Medium (registered 5+ min)"
  );
  console.log(
    "  - Content filter: scan ALL members' messages"
  );
  console.log("  - Default notifications: Only @mentions");
  console.log("  - 2FA required for moderator actions");
  console.log(
    "  - @everyone: no mention-everyone, no invites, no manage"
  );
  console.log(
    "  - AutoMod: spam filter, mass mention block, Discord invite block"
  );
  console.log(
    "  - Slow mode (10s) on help channels"
  );
  console.log(
    "  - Read-only: #ogloszenia, #roadmap, #changelog, #github-feed"
  );
  console.log(
    "  - Maintainers exempt from AutoMod\n"
  );

  console.log("Manual steps remaining:");
  console.log(
    "  1. GitHub webhook for #github-feed (see README.md)"
  );
  console.log(
    "  2. Assign @Maintainer to core team members"
  );
  console.log(
    "  3. Set server icon and banner"
  );
  console.log(
    "  4. Enable Community in Discord settings (optional, for discovery)"
  );

  client.destroy();
}

run().catch((err) => {
  console.error("Setup failed:", err.message);
  client.destroy();
  process.exit(1);
});
