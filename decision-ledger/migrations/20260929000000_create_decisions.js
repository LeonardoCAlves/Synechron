export const up = (pgm) => {
  pgm.createTable('decisions', {
    id: { type: 'uuid', primaryKey: true },
    title: { type: 'varchar(120)', notNull: true },
    status: { type: 'varchar(16)', notNull: true },
    context: { type: 'varchar(4000)', notNull: true },
    decision: { type: 'varchar(4000)', notNull: true },
    consequences: { type: 'varchar(4000)', notNull: true },
    created_at: { type: 'timestamptz', notNull: true },
  });
  pgm.addConstraint('decisions', 'decisions_status_check', {
    check: "status IN ('proposed', 'accepted', 'rejected', 'superseded')",
  });
  pgm.createIndex('decisions', ['status', 'created_at', 'id']);
};

export const down = (pgm) => {
  pgm.dropTable('decisions');
};
