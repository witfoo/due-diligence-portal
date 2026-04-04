-- 003_seed_categories.sql
-- Default document categories for due diligence data rooms.

INSERT OR IGNORE INTO categories (id, name, slug, description, sort_order, icon) VALUES
    ('cat-corporate',   'Corporate',              'corporate',    'Certificate of incorporation, bylaws, org chart, board minutes',                 1, 'Enterprise');
INSERT OR IGNORE INTO categories (id, name, slug, description, sort_order, icon) VALUES
    ('cat-financials',  'Financials',             'financials',   'Financial statements, projections, cap table, tax returns',                      2, 'Finance');
INSERT OR IGNORE INTO categories (id, name, slug, description, sort_order, icon) VALUES
    ('cat-legal',       'Legal',                  'legal',        'Contracts, IP assignments, employment agreements, litigation',                   3, 'Policy');
INSERT OR IGNORE INTO categories (id, name, slug, description, sort_order, icon) VALUES
    ('cat-ip',          'Intellectual Property',  'ip',           'Patents, trademarks, trade secrets, source code documentation',                  4, 'Idea');
INSERT OR IGNORE INTO categories (id, name, slug, description, sort_order, icon) VALUES
    ('cat-team',        'Team',                   'team',         'Key personnel resumes, org structure, hiring plan, compensation',                5, 'Group');
INSERT OR IGNORE INTO categories (id, name, slug, description, sort_order, icon) VALUES
    ('cat-product',     'Product',                'product',      'Product roadmap, architecture, technical documentation, demos',                  6, 'Product');
INSERT OR IGNORE INTO categories (id, name, slug, description, sort_order, icon) VALUES
    ('cat-fundraising', 'Fundraising',            'fundraising',  'Pitch deck, term sheets, investor updates, prior round documents',              7, 'Growth');
INSERT OR IGNORE INTO categories (id, name, slug, description, sort_order, icon) VALUES
    ('cat-compliance',  'Compliance',             'compliance',   'SOC 2, GDPR, regulatory filings, security assessments',                         8, 'TaskComplete');
INSERT OR IGNORE INTO categories (id, name, slug, description, sort_order, icon) VALUES
    ('cat-market',      'Market',                 'market',       'Market research, competitive analysis, customer validation, TAM/SAM/SOM',        9, 'Analytics');
INSERT OR IGNORE INTO categories (id, name, slug, description, sort_order, icon) VALUES
    ('cat-other',       'Other',                  'other',        'Miscellaneous documents',                                                       10, 'Folder');

-- Seed branding and watermark singletons.
INSERT OR IGNORE INTO branding_config (id) VALUES ('default');
INSERT OR IGNORE INTO watermark_config (id) VALUES ('default');
