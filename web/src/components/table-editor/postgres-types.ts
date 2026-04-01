// PostgreSQL data types grouped by category
export const POSTGRES_TYPES: Record<string, string[]> = {
  'Numeric': [
    'smallint',
    'integer',
    'bigint',
    'serial',
    'bigserial',
    'decimal',
    'numeric',
    'real',
    'double precision',
  ],
  'Text': [
    'char',
    'varchar',
    'text',
  ],
  'Date/Time': [
    'date',
    'time',
    'timetz',
    'timestamp',
    'timestamptz',
    'interval',
  ],
  'Boolean': [
    'boolean',
  ],
  'Binary': [
    'bytea',
  ],
  'JSON': [
    'json',
    'jsonb',
  ],
  'UUID': [
    'uuid',
  ],
  'Network': [
    'inet',
    'cidr',
    'macaddr',
  ],
  'Geometric': [
    'point',
    'line',
    'lseg',
    'box',
    'path',
    'polygon',
    'circle',
  ],
}

// Flat list of all types
export const ALL_POSTGRES_TYPES = Object.values(POSTGRES_TYPES).flat()
