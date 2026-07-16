create extension if not exists "pgcrypto";

-- =========================
-- ASESORES
-- =========================

create table asesores (
  id uuid primary key default gen_random_uuid(),
  nombres_completos varchar(150) not null,
  telefono varchar(20),
  email varchar(120),
  activo boolean not null default true,
  created_at timestamp with time zone default now(),
  updated_at timestamp with time zone default now()
);

-- =========================
-- ESTADOS DEL LEAD
-- =========================

create table estado_leads (
  id uuid primary key default gen_random_uuid(),
  nombre varchar(50) not null unique
);

-- =========================
-- PROYECTOS
-- =========================

create table proyectos (
  id uuid primary key default gen_random_uuid(),
  nombre varchar(120) not null,
  tipo varchar(50) not null,
  ubicacion varchar(150),
  estado varchar(50) not null default 'Activo',
  activo boolean not null default true,
  created_at timestamp with time zone default now(),
  updated_at timestamp with time zone default now()
);

-- =========================
-- LEADS
-- =========================

create table leads (
  id uuid primary key default gen_random_uuid(),

  asesor_id uuid,
  estado_lead_id uuid not null,
  proyecto_id uuid,

  etapa_embudo varchar(120) not null default 'Contacto inicial del cliente',
  fuente_prospeccion varchar(50) not null default 'Web',
  lead varchar(50) default 'Cliente',

  nombres_completos varchar(200) not null,
  telefono varchar(20) not null,
  email varchar(120),
  mensaje varchar(300),

  origen_ruta varchar(150),
  origen_componente varchar(120),

  atendido boolean not null default false,
  activo boolean not null default true,

  created_at timestamp with time zone default now(),
  updated_at timestamp with time zone default now(),
  deleted_at timestamp with time zone,

  constraint fk_leads_asesor
    foreign key (asesor_id)
    references asesores(id),

  constraint fk_leads_estado
    foreign key (estado_lead_id)
    references estado_leads(id),

  constraint fk_leads_proyecto
    foreign key (proyecto_id)
    references proyectos(id)
);

-- =========================
-- CLIENTES
-- =========================

create table clientes (
  id uuid primary key default gen_random_uuid(),

  lead_id uuid unique,
  asesor_id uuid,
  proyecto_id uuid,

  nombres_completos varchar(200) not null,
  tipo_documento varchar(20),
  numero_documento varchar(20),
  telefono varchar(20) not null,
  email varchar(120),

  fecha_conversion timestamp with time zone default now(),

  activo boolean not null default true,
  created_at timestamp with time zone default now(),
  updated_at timestamp with time zone default now(),
  deleted_at timestamp with time zone,

  constraint fk_clientes_lead
    foreign key (lead_id)
    references leads(id),

  constraint fk_clientes_asesor
    foreign key (asesor_id)
    references asesores(id),

  constraint fk_clientes_proyecto
    foreign key (proyecto_id)
    references proyectos(id)
);

-- =========================
-- DATOS INICIALES
-- =========================

insert into estado_leads (nombre)
values
  ('Nuevo'),
  ('Contactado'),
  ('Seguimiento'),
  ('Cerrado'),
  ('Perdido');

insert into asesores (nombres_completos, telefono, email)
values
  ('Asesor ANCOSUR', '971069763', 'ventas@ancosur.pe');

insert into proyectos (nombre, tipo, ubicacion, estado, activo)
values
  ('Neo Rivera', 'Departamento', 'Huancayo', 'Activo', true),
  ('Neo Xport', 'Departamento', 'San Carlos', 'Activo', true),
  ('Neo Eterna', 'Departamento', 'San Carlos', 'Activo', true),
  ('Neo Balto', 'Departamento', 'Huancayo', 'Activo', true),
  ('Distrito San Carlos', 'Departamento', 'San Carlos', 'Activo', true),
  ('Camino Real', 'Lote', 'El Tambo', 'Activo', true),
  ('Las Colinas de Moro', 'Lote', 'La Huaycha', 'Activo', true),
  ('Zagari Resort Club', 'Resort', 'San Ramón', 'Activo', true),
  ('Nuevo Resort Oxapampa', 'Resort', 'Oxapampa', 'Próximamente', false);