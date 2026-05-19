-- +goose Up
INSERT INTO asset_types (name, description) VALUES
    ('compressor',       'Gas compressor used in processing or transport'),
    ('tank',             'Storage tank for liquids or gases'),
    ('valve',            'Control or safety valve'),
    ('flare',            'Gas flare stack for combustion of excess gas'),
    ('pump',             'Fluid transfer pump'),
    ('generator',        'Power generation unit'),
    ('heat_exchanger',   'Heat exchange equipment'),
    ('separator',        'Oil/gas/water separation unit'),
    ('pipeline_segment', 'Section of pipeline infrastructure'),
    ('wellhead',         'Surface equipment at a well bore');

-- +goose Down
DELETE FROM asset_types WHERE name IN (
    'compressor', 'tank', 'valve', 'flare', 'pump',
    'generator', 'heat_exchanger', 'separator', 'pipeline_segment', 'wellhead'
);
