-- +goose Up
-- +goose StatementBegin

-- Common OSHA Construction Safety Standards (US)
INSERT INTO safety_codes (code, description, country, state_province) VALUES
-- Fall Protection
('OSHA 1926.501', 'Fall protection requirements for construction - General requirements for fall protection systems', 'US', NULL),
('OSHA 1926.502', 'Fall protection systems criteria and practices', 'US', NULL),
('OSHA 1926.503', 'Training requirements for fall protection', 'US', NULL),

-- Scaffolding
('OSHA 1926.451', 'General requirements for scaffolding construction and use', 'US', NULL),
('OSHA 1926.452', 'Additional requirements applicable to specific types of scaffolds', 'US', NULL),
('OSHA 1926.454', 'Training requirements for scaffold erection, use, and dismantling', 'US', NULL),

-- Personal Protective Equipment (PPE)
('OSHA 1926.95', 'Criteria for personal protective equipment including hard hats, safety glasses, and protective footwear', 'US', NULL),
('OSHA 1926.100', 'Head protection requirements for construction sites', 'US', NULL),
('OSHA 1926.101', 'Hearing protection requirements in high-noise construction environments', 'US', NULL),
('OSHA 1926.102', 'Eye and face protection requirements', 'US', NULL),
('OSHA 1926.103', 'Respiratory protection requirements for construction workers', 'US', NULL),

-- Ladders
('OSHA 1926.1053', 'Ladders - General requirements for ladder safety and proper use', 'US', NULL),
('OSHA 1926.1060', 'Training requirements for ladder use', 'US', NULL),

-- Excavation and Trenching
('OSHA 1926.650', 'Scope and definitions for excavation safety', 'US', NULL),
('OSHA 1926.651', 'Specific excavation requirements including protective systems', 'US', NULL),
('OSHA 1926.652', 'Requirements for protective systems in excavations', 'US', NULL),

-- Electrical Safety
('OSHA 1926.400', 'Introduction to electrical safety standards', 'US', NULL),
('OSHA 1926.404', 'Wiring design and protection requirements', 'US', NULL),
('OSHA 1926.416', 'General requirements for electrical equipment use', 'US', NULL),
('OSHA 1926.417', 'Lockout and tagging of circuits', 'US', NULL),

-- Cranes and Hoisting
('OSHA 1926.1400', 'Scope and definitions for cranes and derricks in construction', 'US', NULL),
('OSHA 1926.1401', 'Signal person qualifications', 'US', NULL),
('OSHA 1926.1404', 'Assembly and disassembly of cranes - General requirements', 'US', NULL),
('OSHA 1926.1427', 'Operator qualification and certification requirements', 'US', NULL),

-- Hazard Communication
('OSHA 1926.59', 'Hazard communication standard for construction - Chemical safety information', 'US', NULL),

-- Confined Spaces
('OSHA 1926.1200', 'Scope and definitions for confined spaces in construction', 'US', NULL),
('OSHA 1926.1201', 'General requirements for confined space entry', 'US', NULL),
('OSHA 1926.1203', 'Permit-required confined spaces', 'US', NULL),
('OSHA 1926.1204', 'Permit system for confined space entry', 'US', NULL),

-- Fire Protection and Prevention
('OSHA 1926.150', 'Fire protection requirements on construction sites', 'US', NULL),
('OSHA 1926.151', 'Fire prevention measures', 'US', NULL),
('OSHA 1926.152', 'Flammable liquids storage and handling', 'US', NULL),

-- Stairways and Walking/Working Surfaces
('OSHA 1926.1050', 'Scope and definitions for stairways and ladders', 'US', NULL),
('OSHA 1926.1051', 'General requirements for stairways', 'US', NULL),
('OSHA 1926.1052', 'Stairways construction requirements', 'US', NULL),

-- Steel Erection
('OSHA 1926.750', 'Scope and definitions for steel erection', 'US', NULL),
('OSHA 1926.751', 'Definitions specific to steel erection', 'US', NULL),
('OSHA 1926.760', 'Fall protection requirements for steel erection', 'US', NULL),

-- Demolition
('OSHA 1926.850', 'Preparatory operations before demolition', 'US', NULL),
('OSHA 1926.851', 'Stairs, passageways, and ladders during demolition', 'US', NULL),
('OSHA 1926.852', 'Chutes for debris removal', 'US', NULL),

-- Concrete and Masonry Construction
('OSHA 1926.700', 'Scope and definitions for concrete and masonry construction', 'US', NULL),
('OSHA 1926.701', 'General requirements for concrete and masonry work', 'US', NULL),
('OSHA 1926.702', 'Requirements for equipment and tools', 'US', NULL),
('OSHA 1926.703', 'Requirements for cast-in-place concrete', 'US', NULL);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM safety_codes WHERE country = 'US' AND code LIKE 'OSHA%';
-- +goose StatementEnd
