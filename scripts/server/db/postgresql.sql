--
-- PostgreSQL database dump
--

-- Dumped from database version 10.17
-- Dumped by pg_dump version 13.3

-- Started on 2021-06-05 10:24:59

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- TOC entry 3 (class 2615 OID 2200)
-- Name: public; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA public;


ALTER SCHEMA public OWNER TO postgres;

--
-- TOC entry 3691 (class 0 OID 0)
-- Dependencies: 3
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: postgres
--

COMMENT ON SCHEMA public IS 'standard public schema';


SET default_tablespace = '';

--
-- TOC entry 197 (class 1259 OID 16491)
-- Name: branch_table; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.branch_table (
    branch_id bigint NOT NULL,
    xid character varying(128) NOT NULL,
    transaction_id bigint,
    resource_group_id character varying(32),
    resource_id character varying(256),
    branch_type character varying(8),
    status integer,
    client_id character varying(64),
    application_data character varying(2000),
    gmt_create timestamp without time zone,
    gmt_modified timestamp without time zone
);


ALTER TABLE public.branch_table OWNER TO postgres;

--
-- TOC entry 196 (class 1259 OID 16479)
-- Name: global_table; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.global_table (
    xid character varying(128) NOT NULL,
    transaction_id bigint,
    status integer NOT NULL,
    application_id character varying(32),
    transaction_service_group character varying(32),
    transaction_name character varying(128),
    timeout integer,
    begin_time bigint,
    application_data character varying(2000),
    gmt_create timestamp without time zone,
    gmt_modified timestamp without time zone
);


ALTER TABLE public.global_table OWNER TO postgres;

--
-- TOC entry 198 (class 1259 OID 16501)
-- Name: lock_table; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.lock_table (
    row_key character varying(128) NOT NULL,
    xid character varying(96),
    transaction_id bigint,
    branch_id bigint NOT NULL,
    resource_id character varying(256),
    table_name character varying(32),
    pk character varying(36),
    gmt_create timestamp without time zone,
    gmt_modified timestamp without time zone
);


ALTER TABLE public.lock_table OWNER TO postgres;

--
-- TOC entry 3684 (class 0 OID 16491)
-- Dependencies: 197
-- Data for Name: branch_table; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.branch_table (branch_id, xid, transaction_id, resource_group_id, resource_id, branch_type, status, client_id, application_data, gmt_create, gmt_modified) FROM stdin;
\.


--
-- TOC entry 3683 (class 0 OID 16479)
-- Dependencies: 196
-- Data for Name: global_table; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.global_table (xid, transaction_id, status, application_id, transaction_service_group, transaction_name, timeout, begin_time, application_data, gmt_create, gmt_modified) FROM stdin;
\.


--
-- TOC entry 3685 (class 0 OID 16501)
-- Dependencies: 198
-- Data for Name: lock_table; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.lock_table (row_key, xid, transaction_id, branch_id, resource_id, table_name, pk, gmt_create, gmt_modified) FROM stdin;
\.


--
-- TOC entry 3555 (class 2606 OID 16498)
-- Name: branch_table branch_table_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.branch_table
    ADD CONSTRAINT branch_table_pkey PRIMARY KEY (branch_id);


--
-- TOC entry 3549 (class 2606 OID 16486)
-- Name: global_table global_table_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.global_table
    ADD CONSTRAINT global_table_pkey PRIMARY KEY (xid);


--
-- TOC entry 3559 (class 2606 OID 16510)
-- Name: lock_table idx_branch_id; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.lock_table
    ADD CONSTRAINT idx_branch_id UNIQUE (branch_id);


--
-- TOC entry 3551 (class 2606 OID 16488)
-- Name: global_table idx_gmt_modified_status; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.global_table
    ADD CONSTRAINT idx_gmt_modified_status UNIQUE (gmt_modified, status);


--
-- TOC entry 3553 (class 2606 OID 16490)
-- Name: global_table idx_transaction_id; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.global_table
    ADD CONSTRAINT idx_transaction_id UNIQUE (transaction_id);


--
-- TOC entry 3557 (class 2606 OID 16500)
-- Name: branch_table idx_xid; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.branch_table
    ADD CONSTRAINT idx_xid UNIQUE (xid);


--
-- TOC entry 3561 (class 2606 OID 16508)
-- Name: lock_table lock_table_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.lock_table
    ADD CONSTRAINT lock_table_pkey PRIMARY KEY (row_key);


-- Completed on 2021-06-05 10:24:59

--
-- PostgreSQL database dump complete
--

