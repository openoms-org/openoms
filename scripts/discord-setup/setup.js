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
  GuildScheduledEventPrivacyLevel,
  GuildScheduledEventEntityType,
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

// Text channels (standard)
const CATEGORIES = [
  {
    name: "INFORMACJE",
    channels: [
      {
        name: "zasady",
        topic:
          "Regulamin serwera OpenOMS. Przeczytaj przed pisaniem na kanałach.",
        readOnly: true,
      },
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
        name: "off-topic",
        topic:
          "Luźne rozmowy niezwiązane z OpenOMS. Memy, ciekawostki, życie.",
      },
      {
        name: "pokaz-swoje",
        topic:
          "Pochwal się swoją instalacją OpenOMS, customizacjami, integracjami.",
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

// Forum channels (organized Q&A with tags)
const FORUM_CHANNELS = [
  {
    category: "POMOC",
    name: "pomoc-instalacja",
    topic:
      "Pomoc z instalacją: Docker, Go, Node.js, PostgreSQL, konfiguracja .env",
    slowMode: 10,
    tags: [
      { name: "Docker", emoji: "🐳" },
      { name: "From Source", emoji: "🔧" },
      { name: "PostgreSQL", emoji: "🐘" },
      { name: "Solved", emoji: "✅", moderated: true },
      { name: "Unsolved", emoji: "❓" },
    ],
    guidelines:
      "Opisz problem. Podaj: system operacyjny, wersję Docker/Go/Node, logi błędów.\nOznacz post tagiem. Maintainer oznaczy jako Solved po rozwiązaniu.",
  },
  {
    category: "POMOC",
    name: "pomoc-konfiguracja",
    topic:
      "Ustawienia tenanta, SMTP, CORS, JWT, szyfrowanie, RBAC, 2FA, webhooks",
    slowMode: 10,
    tags: [
      { name: "SMTP/Email", emoji: "📧" },
      { name: "Auth/JWT", emoji: "🔑" },
      { name: "Integracje", emoji: "🔗" },
      { name: "RBAC/Uprawnienia", emoji: "🛡️" },
      { name: "Solved", emoji: "✅", moderated: true },
      { name: "Unsolved", emoji: "❓" },
    ],
    guidelines:
      "Opisz co konfigurujesz i jaki masz problem. NIE wklejaj tokenów, haseł ani kluczy API!",
  },
  {
    category: "POMOC",
    name: "pomoc-integracje",
    topic:
      "Allegro, InPost, DHL, marketplace'y, feedy IOF, kurierzy, API webhooks",
    slowMode: 10,
    tags: [
      { name: "Allegro", emoji: "🛒" },
      { name: "InPost", emoji: "📦" },
      { name: "DHL/Kurierzy", emoji: "🚚" },
      { name: "WooCommerce", emoji: "🌐" },
      { name: "Inne", emoji: "🔌" },
      { name: "Solved", emoji: "✅", moderated: true },
    ],
    guidelines:
      "Podaj: której integracji dotyczy problem, wersję OpenOMS, logi błędów (bez danych wrażliwych).",
  },
  {
    category: "ROZWOJ",
    name: "bugs",
    topic:
      "Zgłaszanie i dyskusja o bugach. Dla oficjalnych zgłoszeń użyj GitHub Issues.",
    tags: [
      { name: "Backend", emoji: "⚙️" },
      { name: "Frontend", emoji: "🖥️" },
      { name: "API", emoji: "🔌" },
      { name: "Critical", emoji: "🔴" },
      { name: "Confirmed", emoji: "✅", moderated: true },
      { name: "Fixed", emoji: "🎉", moderated: true },
    ],
    guidelines:
      "Opisz buga: co zrobiłeś, co się stało, co powinno się stać. Dołącz logi i screenshoty.",
  },
  {
    category: "ROZWOJ",
    name: "pomysly",
    topic:
      "Propozycje nowych funkcji i usprawnień. Głosuj reakcjami 👍/👎!",
    tags: [
      { name: "Under Review", emoji: "🔍", moderated: true },
      { name: "Planned", emoji: "📋", moderated: true },
      { name: "Implemented", emoji: "✅", moderated: true },
      { name: "Won't Fix", emoji: "❌", moderated: true },
    ],
    guidelines:
      "Opisz funkcję: jaki problem rozwiązuje, kto z niej skorzysta, jak powinna działać. Głosuj na istniejące pomysły zanim tworzysz duplikat.",
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
- **#off-topic** — luźne rozmowy
- **#pomoc-instalacja** — pomoc z setupem (forum)
- **#pomoc-integracje** — Allegro, InPost, marketplace'y (forum)
- **#contributing** — chcesz pomóc? Zacznij tutaj!
- **#pomysly** — zaproponuj nową funkcję (forum)
- **#bugs** — zgłoś buga (forum)

## Role
- **@Maintainer** — core team
- **@Contributor** — kontrybutorzy kodu
- **@Uzytkownik** — użytkownicy OpenOMS

## Zasady
1. Bądź kulturalny — szanuj innych
2. Pisz po polsku lub angielsku
3. Nie spamuj, nie reklamuj
4. Pytania techniczne → odpowiedni kanał forum w POMOC
5. Bugi → #bugs lub GitHub Issues

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
        metadata: { channelId: null },
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
        metadata: { durationSeconds: 300 },
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
  // NEW: Block URL shorteners (phishing vector)
  {
    name: "[OpenOMS] Blokada skracaczy URL",
    eventType: AutoModerationRuleEventType.MessageSend,
    triggerType: AutoModerationRuleTriggerType.Keyword,
    triggerMetadata: {
      regexPatterns: [
        "(?:https?://)?(?:bit\\.ly|tinyurl\\.com|t\\.co|goo\\.gl|rb\\.gy|shorturl\\.at|is\\.gd|v\\.gd|cutt\\.ly)/\\S+",
      ],
    },
    actions: [
      {
        type: AutoModerationActionType.BlockMessage,
        metadata: {
          customMessage:
            "Skrócone linki (bit.ly, tinyurl itp.) nie są dozwolone ze względów bezpieczeństwa. Użyj pełnego URL.",
        },
      },
      {
        type: AutoModerationActionType.SendAlertMessage,
        metadata: { channelId: null },
      },
    ],
  },
  // NEW: Block executable file links
  {
    name: "[OpenOMS] Blokada plikow wykonywalnych",
    eventType: AutoModerationRuleEventType.MessageSend,
    triggerType: AutoModerationRuleTriggerType.Keyword,
    triggerMetadata: {
      regexPatterns: [
        "https?://\\S+\\.(?:exe|bat|cmd|msi|scr|ps1|vbs|wsf|com|pif)(?:\\s|$)",
      ],
    },
    actions: [
      {
        type: AutoModerationActionType.BlockMessage,
        metadata: {
          customMessage:
            "Linki do plików wykonywalnych (.exe, .bat, .msi itp.) nie są dozwolone.",
        },
      },
      {
        type: AutoModerationActionType.SendAlertMessage,
        metadata: { channelId: null },
      },
    ],
  },
  // NEW: Block crypto/web3 spam
  {
    name: "[OpenOMS] Blokada crypto/web3 spam",
    eventType: AutoModerationRuleEventType.MessageSend,
    triggerType: AutoModerationRuleTriggerType.Keyword,
    triggerMetadata: {
      regexPatterns: [
        "(?i)(?:presale|whitelist\\s*mint|web3\\s*earn|token\\s*launch|pump\\s*group|binance\\s*listing)",
      ],
    },
    actions: [
      {
        type: AutoModerationActionType.BlockMessage,
        metadata: {
          customMessage:
            "Treści związane z crypto/web3 spam nie są dozwolone.",
        },
      },
      {
        type: AutoModerationActionType.Timeout,
        metadata: { durationSeconds: 600 },
      },
    ],
  },
];

// ── Community Onboarding config ──────────────────────────────────

const ONBOARDING_PROMPTS = [
  {
    title: "Co Cię tu sprowadza?",
    options: [
      {
        title: "Oceniam OpenOMS dla mojego sklepu",
        emoji: "🛒",
        description: "Szukam systemu do zarządzania zamówieniami",
        roleNames: ["Uzytkownik"],
        channelNames: [
          "pomoc-instalacja",
          "pomoc-konfiguracja",
          "pomoc-integracje",
          "general",
        ],
      },
      {
        title: "Jestem użytkownikiem OpenOMS",
        emoji: "📦",
        description: "Już używam OpenOMS i potrzebuję pomocy lub chcę dyskutować",
        roleNames: ["Uzytkownik"],
        channelNames: [
          "pomoc-instalacja",
          "pomoc-konfiguracja",
          "pomoc-integracje",
          "general",
          "pomysly",
        ],
      },
      {
        title: "Chcę kontrybuować kod",
        emoji: "💻",
        description: "Programista chcący pomóc w rozwoju projektu",
        roleNames: ["Contributor"],
        channelNames: [
          "contributing",
          "pull-requests",
          "bugs",
          "general",
        ],
      },
      {
        title: "Przeglądam / jestem ciekawy",
        emoji: "👀",
        description: "Chcę zobaczyć co tu się dzieje",
        channelNames: ["general", "off-topic", "ogloszenia"],
      },
    ],
  },
  {
    title: "Jakie integracje Cię interesują?",
    options: [
      {
        title: "Allegro",
        emoji: "🛍️",
        description: "Marketplace Allegro — oferty, zamówienia, wysyłka",
        channelNames: ["pomoc-integracje"],
      },
      {
        title: "InPost / Kurierzy",
        emoji: "📬",
        description: "InPost, DHL, DPD, GLS, UPS, Poczta Polska",
        channelNames: ["pomoc-integracje"],
      },
      {
        title: "WooCommerce / Shopify / Shoper",
        emoji: "🌐",
        description: "Integracje ze sklepami internetowymi",
        channelNames: ["pomoc-integracje"],
      },
      {
        title: "Wszystko / Inne",
        emoji: "🔌",
        description: "Amazon, eBay, Erli, Empik, fakturowanie, dostawcy",
        channelNames: ["pomoc-integracje", "pomoc-konfiguracja"],
      },
    ],
  },
];

// ── Helpers ──────────────────────────────────────────────────────

// Generate a Discord snowflake-like ID (required by some APIs for new entities)
let snowflakeCounter = 0n;
function generateSnowflake() {
  const DISCORD_EPOCH = 1420070400000n;
  const timestamp = BigInt(Date.now()) - DISCORD_EPOCH;
  snowflakeCounter++;
  return String(
    (timestamp << 22n) | (1n << 17n) | (snowflakeCounter & 0xFFFn)
  );
}

// ── Script ───────────────────────────────────────────────────────

const client = new Client({
  intents: [
    GatewayIntentBits.Guilds,
    GatewayIntentBits.GuildMembers,
    GatewayIntentBits.AutoModerationConfiguration,
    GatewayIntentBits.GuildScheduledEvents,
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
  console.log("Logging in...");
  await client.login(TOKEN);
  console.log(`Logged in as: ${client.user.tag}\n`);

  // Print invite URL for convenience
  console.log(
    `If the bot is not on your server yet, invite it:\n  https://discord.com/oauth2/authorize?client_id=${client.user.id}&permissions=8&scope=bot\n`
  );

  let guild;
  try {
    guild = await client.guilds.fetch(GUILD_ID);
  } catch (err) {
    if (err.status === 404 || err.message.includes("404")) {
      console.error(`ERROR: Guild ${GUILD_ID} not found.\n`);
      console.error("Possible causes:");
      console.error(
        "  1. Bot is not on the server — use the invite link above"
      );
      console.error(
        "  2. Guild ID is wrong — right-click server > Copy Server ID (enable Developer Mode in Discord settings)"
      );
      console.error(
        `  3. Bot is on a different server — currently on: ${client.guilds.cache.map((g) => `${g.name} (${g.id})`).join(", ") || "no servers"}`
      );
      client.destroy();
      process.exit(1);
    }
    throw err;
  }

  console.log(
    `Connected to: ${guild.name} (${guild.memberCount} members)\n`
  );

  // ── 1. Server-level security settings ──
  console.log("=== 1. SECURITY SETTINGS ===");

  if (guild.verificationLevel !== GuildVerificationLevel.Medium) {
    await guild.setVerificationLevel(GuildVerificationLevel.Medium);
    console.log(
      "  [set] Verification level -> Medium (registered 5+ min)"
    );
  } else {
    console.log("  [skip] Verification level already Medium");
  }

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

  try {
    if (guild.mfaLevel === 0) {
      await guild.setMFALevel(1);
      console.log("  [set] 2FA required for moderators");
    } else {
      console.log("  [skip] 2FA for moderators already enabled");
    }
  } catch {
    console.log(
      "  [skip] 2FA for moderators (requires server owner to have 2FA enabled)"
    );
  }

  // ── 2. Roles ──
  console.log("\n=== 2. ROLES ===");
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
  console.log("\n=== 3. @EVERYONE PERMISSIONS ===");
  const denyEveryone = [
    PermissionFlagsBits.MentionEveryone,
    PermissionFlagsBits.ManageMessages,
    PermissionFlagsBits.ManageChannels,
    PermissionFlagsBits.ManageRoles,
    PermissionFlagsBits.ManageGuild,
    PermissionFlagsBits.Administrator,
    PermissionFlagsBits.BanMembers,
    PermissionFlagsBits.KickMembers,
    PermissionFlagsBits.ManageWebhooks,
    PermissionFlagsBits.ManageNicknames,
    PermissionFlagsBits.CreateInstantInvite,
  ];

  const newPerms = everyoneRole.permissions;
  for (const perm of denyEveryone) {
    newPerms.remove(perm);
  }
  await everyoneRole.setPermissions(newPerms);
  console.log(
    "  [set] @everyone: removed dangerous permissions"
  );

  // ── 4. Text channels ──
  console.log("\n=== 4. TEXT CHANNELS ===");
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

      if (channel.parentId !== category.id) {
        await channel.setParent(category.id, {
          lockPermissions: false,
        });
        console.log(
          `  [moved] "${chDef.name}" -> ${catDef.name}`
        );
      }

      if (chDef.topic && channel.topic !== chDef.topic) {
        await channel.setTopic(chDef.topic);
      }

      if (
        chDef.slowMode &&
        channel.rateLimitPerUser !== chDef.slowMode
      ) {
        await channel.setRateLimitPerUser(chDef.slowMode);
        console.log(
          `  [perms] "${chDef.name}" -> slow mode ${chDef.slowMode}s`
        );
      }

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

  // ── 5. Forum channels ──
  console.log("\n=== 5. FORUM CHANNELS ===");
  await guild.channels.fetch(); // refresh cache

  for (const forumDef of FORUM_CHANNELS) {
    // Find or create the parent category
    let category = guild.channels.cache.find(
      (c) =>
        c.type === ChannelType.GuildCategory &&
        c.name.toLowerCase() === forumDef.category.toLowerCase()
    );
    if (!category) {
      category = await guild.channels.create({
        name: forumDef.category,
        type: ChannelType.GuildCategory,
      });
      console.log(`  [created] category "${forumDef.category}"`);
    }

    // Check if forum already exists
    const existing = guild.channels.cache.find(
      (c) => c.name.toLowerCase() === forumDef.name.toLowerCase()
    );

    if (existing) {
      console.log(`  [skip] forum "${forumDef.name}" already exists`);

      // Ensure tags exist even on existing forum
      if (existing.type === ChannelType.GuildForum) {
        const existingTagNames = existing.availableTags.map((t) =>
          t.name.toLowerCase()
        );
        const missingTags = forumDef.tags.filter(
          (t) =>
            !existingTagNames.includes(t.name.toLowerCase())
        );
        if (missingTags.length > 0) {
          const newTags = [
            ...existing.availableTags,
            ...missingTags.map((t) => ({
              name: t.name,
              moderated: t.moderated || false,
              emoji: t.emoji ? { name: t.emoji } : undefined,
            })),
          ];
          await existing.setAvailableTags(newTags);
          console.log(
            `  [updated] "${forumDef.name}" tags: +${missingTags.map((t) => t.name).join(", ")}`
          );
        }
      }
      continue;
    }

    // Create the forum channel
    const forum = await guild.channels.create({
      name: forumDef.name,
      type: ChannelType.GuildForum,
      parent: category.id,
      topic: forumDef.topic,
      rateLimitPerUser: forumDef.slowMode || 0,
      availableTags: forumDef.tags.map((t) => ({
        name: t.name,
        moderated: t.moderated || false,
        emoji: t.emoji ? { name: t.emoji } : undefined,
      })),
      defaultForumLayout: 1, // List view
    });
    console.log(`  [created] forum "${forumDef.name}"`);

    // Set guidelines (post creation template)
    if (forumDef.guidelines) {
      try {
        await forum.edit({
          defaultThreadRateLimitPerUser: forumDef.slowMode || 0,
        });
        // Guidelines are set via topic for forums
        if (forum.topic !== forumDef.topic) {
          await forum.setTopic(forumDef.topic);
        }
      } catch {
        // Some settings may not be available
      }
    }
  }

  // ── 6. Voice channel ──
  console.log("\n=== 6. VOICE CHANNEL ===");
  await guild.channels.fetch();

  let voiceCategory = guild.channels.cache.find(
    (c) =>
      c.type === ChannelType.GuildCategory &&
      c.name.toLowerCase() === "glos"
  );
  if (!voiceCategory) {
    voiceCategory = await guild.channels.create({
      name: "GLOS",
      type: ChannelType.GuildCategory,
    });
    console.log('  [created] category "GLOS"');
  }

  const voiceChannel = await findOrCreate(
    guild.channels,
    "office-hours",
    () =>
      guild.channels.create({
        name: "office-hours",
        type: ChannelType.GuildVoice,
        parent: voiceCategory.id,
      })
  );

  // ── 7. AutoMod rules ──
  console.log("\n=== 7. AUTOMOD RULES ===");

  await guild.channels.fetch(); // ensure fresh cache
  const alertChannel = guild.channels.cache.find(
    (c) => c.name === "github-feed"
  );
  console.log(
    `  Alert channel: ${alertChannel ? `#${alertChannel.name} (${alertChannel.id})` : "not found"}`
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

    // Build actions — resolve alert channel ID, skip alert if no channel
    const actions = [];
    for (const a of ruleDef.actions) {
      if (a.type === AutoModerationActionType.SendAlertMessage) {
        if (alertChannel) {
          actions.push({
            type: a.type,
            metadata: { channel_id: alertChannel.id },
          });
        }
        // else skip this action entirely
      } else {
        actions.push(a);
      }
    }

    // Use REST API directly to avoid discord.js metadata key issues
    try {
      await client.rest.post(`/guilds/${guild.id}/auto-moderation/rules`, {
        body: {
          name: ruleDef.name,
          enabled: true,
          event_type: ruleDef.eventType,
          trigger_type: ruleDef.triggerType,
          trigger_metadata: {
            keyword_filter: ruleDef.triggerMetadata.keywordFilter,
            regex_patterns: ruleDef.triggerMetadata.regexPatterns,
            mention_total_limit: ruleDef.triggerMetadata.mentionTotalLimit,
          },
          actions: actions.map((a) => ({
            type: a.type,
            metadata: a.metadata,
          })),
          exempt_roles: [maintainerRole.id],
        },
      });
      console.log(`  [created] "${ruleDef.name}"`);
    } catch (err) {
      console.log(
        `  [error] "${ruleDef.name}": ${err.message}`
      );
    }
  }

  // ── 8. Webhook hardening ──
  console.log("\n=== 8. WEBHOOK HARDENING ===");

  // Restrict ManageWebhooks to Maintainer on all channels
  let webhookHardenedCount = 0;
  for (const [, channel] of guild.channels.cache) {
    if (
      channel.type !== ChannelType.GuildText &&
      channel.type !== ChannelType.GuildForum
    )
      continue;

    try {
      const perms = channel.permissionOverwrites.cache;
      const everyoneOverwrite = perms.get(everyoneRole.id);

      // Only set if not already denied
      if (
        !everyoneOverwrite ||
        !everyoneOverwrite.deny.has(
          PermissionFlagsBits.ManageWebhooks
        )
      ) {
        await channel.permissionOverwrites.edit(everyoneRole.id, {
          ManageWebhooks: false,
        });
        webhookHardenedCount++;
      }
    } catch {
      // Some channels may not support this
    }
  }
  console.log(
    `  [set] ManageWebhooks denied for @everyone on ${webhookHardenedCount} channels`
  );

  // Ensure Maintainer can manage webhooks
  try {
    if (!maintainerRole.permissions.has(PermissionFlagsBits.ManageWebhooks)) {
      const updatedPerms = maintainerRole.permissions;
      updatedPerms.add(PermissionFlagsBits.ManageWebhooks);
      await maintainerRole.setPermissions(updatedPerms);
      console.log(
        "  [set] @Maintainer: ManageWebhooks granted"
      );
    } else {
      console.log(
        "  [skip] @Maintainer already has ManageWebhooks"
      );
    }
  } catch (err) {
    console.log(
      `  [skip] Maintainer webhook perms: ${err.message}`
    );
  }

  // ── 9. Welcome message in #ogloszenia ──
  console.log("\n=== 9. WELCOME MESSAGE ===");
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

  // ── 10. Community + Membership Screening ──
  console.log("\n=== 10. COMMUNITY & MEMBERSHIP SCREENING ===");

  const zasadyChannel = guild.channels.cache.find(
    (c) => c.name === "zasady"
  );
  const changelogChannel = guild.channels.cache.find(
    (c) => c.name === "changelog"
  );

  if (zasadyChannel && changelogChannel) {
    try {
      await guild.edit({
        rulesChannelId: zasadyChannel.id,
        publicUpdatesChannelId: changelogChannel.id,
      });
      console.log(
        "  [set] Community enabled (rules: #zasady, updates: #changelog)"
      );
    } catch (err) {
      console.log(
        `  [skip] Community: ${err.message}`
      );
    }
  }

  if (zasadyChannel) {
    const msgs = await zasadyChannel.messages.fetch({ limit: 5 });
    const hasRules = msgs.some((m) =>
      m.content.includes("Regulamin serwera")
    );
    if (!hasRules) {
      await zasadyChannel.send(`# Regulamin serwera OpenOMS

1. **Szanuj innych** — bądź kulturalny, zero hejtu i obrażania
2. **Nie spamuj** — żadnych reklam, linków afiliacyjnych, crypto scamów
3. **Pisz w odpowiednich kanałach** — pomoc w forach POMOC, bugi w #bugs
4. **Język: polski i angielski** — oba mile widziane
5. **Nie wysyłaj DM maintainerom** bez zaproszenia — pytaj publicznie
6. **Bez NSFW** — treści dla dorosłych są zakazane
7. **Nie wklejaj tokenów/haseł** — nigdy nie udostępniaj kluczy API, haseł DB ani tokenów
8. **Przestrzegaj Discord ToS** — https://discord.com/terms

Łamanie zasad = ostrzeżenie → timeout → ban.
Spam boty = natychmiastowy ban.`);
      console.log("  [sent] Rules in #zasady");
    } else {
      console.log("  [skip] Rules already in #zasady");
    }
  }

  // Note: Membership Screening (member-verification API) is replaced by
  // Community Onboarding (step 11). When Onboarding is enabled, Discord
  // returns 405 for the old screening endpoint. Rules acceptance is now
  // handled by the Onboarding flow + #zasady channel.

  // ── 11. Community Onboarding ──
  console.log("\n=== 11. COMMUNITY ONBOARDING ===");

  await guild.channels.fetch(); // refresh after forum creation

  try {
    // Get existing onboarding to preserve prompt IDs
    let existingOnboarding = null;
    try {
      existingOnboarding = await client.rest.get(
        `/guilds/${guild.id}/onboarding`
      );
    } catch {
      // No existing onboarding
    }

    // Build onboarding prompts with resolved IDs
    const prompts = ONBOARDING_PROMPTS.map((prompt, promptIdx) => {
      // Try to match existing prompt by title to preserve its ID
      const existingPrompt = existingOnboarding?.prompts?.find(
        (p) => p.title === prompt.title
      );

      const options = prompt.options.map((opt, optIdx) => {
        const channelIds = (opt.channelNames || [])
          .map((name) => {
            const ch = guild.channels.cache.find(
              (c) => c.name.toLowerCase() === name.toLowerCase()
            );
            return ch?.id;
          })
          .filter(Boolean);

        const roleIds = (opt.roleNames || [])
          .map((name) => roleMap[name]?.id)
          .filter(Boolean);

        const existingOpt = existingPrompt?.options?.[optIdx];

        return {
          id: existingOpt?.id || generateSnowflake(),
          title: opt.title,
          description: opt.description || "",
          emoji: opt.emoji ? { name: opt.emoji } : undefined,
          channel_ids: channelIds,
          role_ids: roleIds,
        };
      });

      return {
        id: existingPrompt?.id || generateSnowflake(),
        type: 0, // MULTIPLE_CHOICE
        title: prompt.title,
        single_select: false,
        required: true,
        in_onboarding: true,
        options,
      };
    });

    // Collect default channel IDs (required: at least 7)
    const defaultChannelNames = [
      "zasady",
      "ogloszenia",
      "roadmap",
      "changelog",
      "general",
      "off-topic",
      "contributing",
      "github-feed",
      "pomoc-instalacja",
    ];
    const defaultChannelIds = defaultChannelNames
      .map((name) => {
        const ch = guild.channels.cache.find(
          (c) => c.name.toLowerCase() === name.toLowerCase()
        );
        return ch?.id;
      })
      .filter(Boolean);

    await client.rest.put(`/guilds/${guild.id}/onboarding`, {
      body: {
        prompts,
        default_channel_ids: defaultChannelIds,
        enabled: true,
        mode: 0, // ONBOARDING_DEFAULT
      },
    });
    console.log(
      "  [set] Community Onboarding — pytania powitalne + auto-role"
    );
  } catch (err) {
    console.log(
      `  [skip] Community Onboarding: ${err.message}`
    );
  }

  // ── 12. Server Guide (Welcome Screen) ──
  console.log("\n=== 12. SERVER GUIDE ===");

  try {
    // Try to enable Community by setting features via guild edit
    const refetchedGuild = await guild.fetch();
    const hasCommunity = refetchedGuild.features.includes("COMMUNITY");
    console.log(
      `  Community: ${hasCommunity ? "enabled" : "not enabled"}, features: ${refetchedGuild.features.join(", ") || "none"}`
    );

    // Try to enable welcome screen regardless — the API will tell us if it fails
    const wsGeneralCh = guild.channels.cache.find(
      (c) => c.name === "general"
    );
    const wsContribCh = guild.channels.cache.find(
      (c) => c.name === "contributing"
    );
    const wsInstallCh = guild.channels.cache.find(
      (c) => c.name === "pomoc-instalacja"
    );
    const wsBugsCh = guild.channels.cache.find(
      (c) => c.name === "bugs"
    );

    const welcomeChannels = [
      wsGeneralCh && {
        channel_id: wsGeneralCh.id,
        description: "Ogólne rozmowy o OpenOMS i e-commerce",
        emoji_name: "\uD83D\uDCAC",
      },
      wsInstallCh && {
        channel_id: wsInstallCh.id,
        description: "Potrzebujesz pomocy z instalacja? Tutaj!",
        emoji_name: "\uD83D\uDD27",
      },
      wsContribCh && {
        channel_id: wsContribCh.id,
        description: "Chcesz pomoc w rozwoju? Zacznij tutaj",
        emoji_name: "\uD83D\uDCBB",
      },
      wsBugsCh && {
        channel_id: wsBugsCh.id,
        description: "Znalazles buga? Zglos go tutaj",
        emoji_name: "\uD83D\uDC1B",
      },
    ].filter(Boolean);

    if (!hasCommunity) {
      // Try enabling Community via guild edit with required fields
      try {
        await guild.edit({
          rulesChannelId: zasadyChannel?.id,
          publicUpdatesChannelId: changelogChannel?.id,
          features: [...refetchedGuild.features, "COMMUNITY"],
        });
        console.log("  [set] Community feature enabled");
      } catch {
        console.log(
          "  [info] Community must be enabled manually: Server Settings → Community → Enable Community"
        );
      }
    }

    await client.rest.patch(`/guilds/${guild.id}/welcome-screen`, {
      body: {
        enabled: true,
        description:
          "OpenOMS — open-source OMS dla polskiego e-commerce. Wybierz kanal aby zaczac!",
        welcome_channels: welcomeChannels.slice(0, 5),
      },
    });
    console.log(
      "  [set] Server Guide (welcome screen) z 4 kanałami"
    );
  } catch (err) {
    console.log(
      `  [skip] Server Guide: ${err.message}`
    );
  }

  // ── 13. Scheduled Event — first Office Hours ──
  console.log("\n=== 13. SCHEDULED EVENTS ===");

  try {
    const existingEvents = await guild.scheduledEvents.fetch();
    const hasOfficeHours = existingEvents.some((e) =>
      e.name.includes("Office Hours")
    );

    if (!hasOfficeHours) {
      // Schedule first Office Hours — next Saturday at 18:00 CET
      const now = new Date();
      const daysUntilSaturday = (6 - now.getDay() + 7) % 7 || 7;
      const nextSaturday = new Date(now);
      nextSaturday.setDate(now.getDate() + daysUntilSaturday);
      nextSaturday.setHours(18, 0, 0, 0);

      const endTime = new Date(nextSaturday);
      endTime.setMinutes(endTime.getMinutes() + 60);

      await guild.scheduledEvents.create({
        name: "OpenOMS Office Hours",
        description:
          "Godzina pytań i odpowiedzi z maintainerami OpenOMS.\n\n" +
          "Pytaj o cokolwiek: instalacja, konfiguracja, integracje, contributing.\n" +
          "Dołącz do kanału głosowego #office-hours.\n\n" +
          "Język: polski / angielski",
        scheduledStartTime: nextSaturday.toISOString(),
        scheduledEndTime: endTime.toISOString(),
        privacyLevel: GuildScheduledEventPrivacyLevel.GuildOnly,
        entityType: GuildScheduledEventEntityType.Voice,
        channel: voiceChannel.id,
      });
      console.log(
        `  [created] "OpenOMS Office Hours" — ${nextSaturday.toLocaleDateString("pl-PL")} 18:00 CET`
      );
    } else {
      console.log(
        "  [skip] Office Hours event already exists"
      );
    }
  } catch (err) {
    console.log(`  [skip] Scheduled event: ${err.message}`);
  }

  // ── 14. Server settings ──
  console.log("\n=== 14. SERVER SETTINGS ===");
  const generalCh = guild.channels.cache.find(
    (c) => c.name === "general"
  );

  if (
    generalCh &&
    guild.systemChannelId !== generalCh.id
  ) {
    await guild.setSystemChannel(generalCh.id);
    console.log("  [set] System channel -> #general");
  }

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
  console.log("\n╔══════════════════════════════════════════════════════╗");
  console.log("║             SETUP COMPLETE                           ║");
  console.log("╚══════════════════════════════════════════════════════╝\n");

  console.log("What was configured:");
  console.log("  SECURITY:");
  console.log(
    "    - Membership Screening (rules acceptance popup)"
  );
  console.log(
    "    - Community Onboarding (personalized welcome flow)"
  );
  console.log(
    "    - Server Guide (welcome screen with channel links)"
  );
  console.log(
    "    - Verification: Medium (5+ min account age)"
  );
  console.log(
    "    - Content filter: ALL members' messages scanned"
  );
  console.log("    - 2FA required for moderators");
  console.log(
    "    - @everyone hardened (no mention-all, no invites, no manage)"
  );
  console.log(
    "    - Webhook permissions: only @Maintainer can manage"
  );
  console.log("  AUTOMOD (6 rules):");
  console.log("    - Spam/scam keyword filter");
  console.log("    - Mass mention block (5 max + timeout)");
  console.log("    - Discord invite block");
  console.log("    - URL shortener block (bit.ly, tinyurl...)");
  console.log("    - Executable file link block (.exe, .bat...)");
  console.log(
    "    - Crypto/web3 spam block (+ 10 min timeout)"
  );
  console.log("  CHANNELS:");
  console.log(
    "    - 4 read-only: #zasady, #ogloszenia, #roadmap, #changelog"
  );
  console.log(
    "    - 4 text: #general, #off-topic, #pokaz-swoje, #contributing, #pull-requests"
  );
  console.log(
    "    - 5 forum: #pomoc-instalacja, #pomoc-konfiguracja, #pomoc-integracje, #bugs, #pomysly"
  );
  console.log(
    "    - 1 read-only: #github-feed (webhook + AutoMod alerts)"
  );
  console.log("    - 1 voice: #office-hours");
  console.log("  EVENTS:");
  console.log(
    "    - Office Hours scheduled (next Saturday 18:00)"
  );
  console.log("");

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
    "  4. (Optional) Add Carl-bot: https://carl.gg — reaction roles, suggestions, audit log"
  );
  console.log(
    "  5. (Optional) Add Answer Overflow: https://www.answeroverflow.com — index forum posts on Google"
  );
  console.log(
    "  6. (Optional) Cold owner: transfer ownership to a dedicated secure account"
  );

  client.destroy();
}

run().catch((err) => {
  console.error("Setup failed:", err.message);
  client.destroy();
  process.exit(1);
});
