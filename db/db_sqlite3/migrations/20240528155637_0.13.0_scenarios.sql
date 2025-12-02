-- +goose Up
CREATE TABLE scenarios (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id BIGINT,
    name VARCHAR(255) NOT NULL,
    description VARCHAR(255) NOT NULL,
    created_date DATETIME,
    modified_date DATETIME,
    page_id BIGINT,
    url VARCHAR(255),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (page_id) REFERENCES pages(id)
);

CREATE TABLE scenario_templates (
    scenario_id INTEGER,
    template_id INTEGER,
    FOREIGN KEY (scenario_id) REFERENCES scenarios(id),
    FOREIGN KEY (template_id) REFERENCES templates(id)
);

CREATE TABLE campaign_scenarios (
    campaign_id INTEGER,
    scenario_id INTEGER,
    FOREIGN KEY (scenario_id) REFERENCES scenarios(id),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id)
);

ALTER TABLE "results" ADD COLUMN "template_id" INTEGER;
ALTER TABLE "results" ADD COLUMN "scenario_id" INTEGER;
ALTER TABLE "mail_logs" ADD COLUMN "template_id" INTEGER;
ALTER TABLE "mail_logs" ADD COLUMN "scenario_id" INTEGER;

-- Insert data into 'scenarios' table based on the 'campaigns' table
INSERT INTO scenarios (id, user_id, name, description, created_date, page_id, url)
SELECT id, user_id, name, 'Auto-generated from campaigns', created_date, page_id, url
FROM campaigns;

-- Insert data into 'scenario_templates' table for migrated template relationships
INSERT INTO scenario_templates (scenario_id, template_id)
SELECT s.id, c.template_id
FROM campaigns c
JOIN scenarios s ON s.id = c.id;

-- Insert data into 'campaign_scenarios' table for campaign-scenario relationships
INSERT INTO campaign_scenarios (campaign_id, scenario_id)
SELECT c.id, s.id
FROM campaigns c
JOIN scenarios s ON s.id = c.id;

-- Populate 'scenario_id' in 'results' based on 'campaign_scenarios'
UPDATE results
SET scenario_id = (
    SELECT cs.scenario_id
    FROM campaign_scenarios cs
    WHERE cs.campaign_id = results.campaign_id
);

-- Populate 'template_id' in 'results' based on 'scenario_templates' and 'campaign_scenarios'
UPDATE results
SET template_id = (
    SELECT st.template_id
    FROM scenario_templates st
    JOIN campaign_scenarios cs ON st.scenario_id = cs.scenario_id
    WHERE cs.campaign_id = results.campaign_id
);

-- Populate 'scenario_id' in 'mail_logs' based on 'campaign_scenarios'
UPDATE mail_logs
SET scenario_id = (
    SELECT cs.scenario_id
    FROM campaign_scenarios cs
    WHERE cs.campaign_id = mail_logs.campaign_id
);

-- Populate 'template_id' in 'mail_logs' based on 'scenario_templates' and 'campaign_scenarios'
UPDATE mail_logs
SET template_id = (
    SELECT st.template_id
    FROM scenario_templates st
    JOIN campaign_scenarios cs ON st.scenario_id = cs.scenario_id
    WHERE cs.campaign_id = mail_logs.campaign_id
);

-- +goose Down
DROP TABLE scenarios;
DROP TABLE scenario_templates;
